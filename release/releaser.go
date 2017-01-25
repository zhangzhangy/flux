package release

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/pkg/errors"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/instance"
	"github.com/weaveworks/flux/jobs"
	fluxmetrics "github.com/weaveworks/flux/metrics"
	"github.com/weaveworks/flux/platform"
	"github.com/weaveworks/flux/platform/kubernetes"
)

const FluxServiceName = "fluxsvc"
const FluxDaemonName = "fluxd"

var (
	// ErrNoop is used to break out of a release early, if there is nothing to
	// do.
	ErrNoop = fmt.Errorf("Nothing to do.")
)

type Releaser struct {
	instancer instance.Instancer
	metrics   Metrics
}

type Metrics struct {
	ReleaseDuration metrics.Histogram
	ActionDuration  metrics.Histogram
	StageDuration   metrics.Histogram
}

func NewReleaser(
	instancer instance.Instancer,
	metrics Metrics,
) *Releaser {
	return &Releaser{
		instancer: instancer,
		metrics:   metrics,
	}
}

type ReleaseAction struct {
	Name        string                                `json:"name"`
	Description string                                `json:"description"`
	Do          func(*ReleaseContext) (string, error) `json:"-"`
	Result      string                                `json:"result"`
}

func (r *Releaser) Handle(job *jobs.Job, updater jobs.JobUpdater) (followUps []jobs.Job, err error) {
	params := job.Params.(jobs.ReleaseJobParams)

	// Backwards compatibility
	if string(params.ServiceSpec) != "" {
		params.ServiceSpecs = append(params.ServiceSpecs, params.ServiceSpec)
	}

	releaseType := "unknown"
	defer func(begin time.Time) {
		r.metrics.ReleaseDuration.With(
			fluxmetrics.LabelReleaseType, releaseType,
			fluxmetrics.LabelReleaseKind, string(params.Kind),
			fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
		).Observe(time.Since(begin).Seconds())
	}(time.Now())

	inst, err := r.instancer.Get(job.Instance)
	if err != nil {
		return nil, err
	}

	inst.Logger = log.NewContext(inst.Logger).With("job", job.ID)

	updateJob := func(format string, args ...interface{}) {
		status := fmt.Sprintf(format, args...)
		job.Status = status
		job.Log = append(job.Log, status)
		updater.UpdateJob(*job)
	}

	var actions []ReleaseAction
	releaseType, actions, err = r.plan(inst, params)
	if err != nil {
		return nil, errors.Wrap(err, "planning release")
	}
	return nil, r.execute(inst, actions, params.Kind, updateJob)
}

func (r *Releaser) plan(inst *instance.Instance, params jobs.ReleaseJobParams) (string, []ReleaseAction, error) {
	releaseType := "unknown"

	images := ImageSelectorForSpec(params.ImageSpec)

	services, err := ServiceSelectorForSpecs(inst, params.ServiceSpecs, params.Excludes)
	if err != nil {
		return releaseType, nil, err
	}

	msg := fmt.Sprintf("Release %v to %v", images, services)
	var actions []ReleaseAction
	switch {
	case params.ServiceSpec == flux.ServiceSpecAll && params.ImageSpec == flux.ImageSpecLatest:
		releaseType = "release_all_to_latest"
		actions, err = r.releaseImages(releaseType, msg, inst, services, images)

	case params.ServiceSpec == flux.ServiceSpecAll && params.ImageSpec == flux.ImageSpecNone:
		releaseType = "release_all_without_update"
		actions, err = r.releaseWithoutUpdate(releaseType, msg, inst, services)

	case params.ServiceSpec == flux.ServiceSpecAll:
		releaseType = "release_all_for_image"
		actions, err = r.releaseImages(releaseType, msg, inst, services, images)

	case params.ImageSpec == flux.ImageSpecLatest:
		releaseType = "release_one_to_latest"
		actions, err = r.releaseImages(releaseType, msg, inst, services, images)

	case params.ImageSpec == flux.ImageSpecNone:
		releaseType = "release_one_without_update"
		actions, err = r.releaseWithoutUpdate(releaseType, msg, inst, services)

	default:
		releaseType = "release_one"
		actions, err = r.releaseImages(releaseType, msg, inst, services, images)
	}
	return releaseType, actions, err
}

func (r *Releaser) releaseImages(method, msg string, inst *instance.Instance, getServices ServiceSelector, getImages ImageSelector) ([]ReleaseAction, error) {
	return []ReleaseAction{
		r.releaseActionPrintf(msg),
		r.releaseActionClone(),
		r.releaseActionFetchPlatformServices(getServices),
		r.releaseActionCheckForNewImages(getImages),
		r.releaseActionFindDefinitions(getServices),
		r.releaseActionUpdateDefinitions(getImages),
		r.releaseActionCommitAndPush(msg),
		r.releaseActionApplyToPlatform(msg),
	}, nil
}

// Release whatever is in the cloned configuration, without changing anything
func (r *Releaser) releaseWithoutUpdate(method, msg string, inst *instance.Instance, getServices ServiceSelector) ([]ReleaseAction, error) {
	return []ReleaseAction{
		r.releaseActionPrintf(msg),
		r.releaseActionClone(),
		r.releaseActionFetchPlatformServices(getServices),
		r.releaseActionFindDefinitions(getServices),
		r.releaseActionApplyToPlatform(msg),
	}, nil
}

func (r *Releaser) execute(inst *instance.Instance, actions []ReleaseAction, kind flux.ReleaseKind, updateJob func(string, ...interface{})) error {
	rc := NewReleaseContext(inst)
	defer rc.Clean()

	for i, action := range actions {
		updateJob(action.Description)
		inst.Log("description", action.Description)
		if action.Do == nil {
			continue
		}

		if kind == flux.ReleaseKindExecute {
			begin := time.Now()
			result, err := action.Do(rc)
			r.metrics.ActionDuration.With(
				fluxmetrics.LabelAction, action.Name,
				fluxmetrics.LabelSuccess, fmt.Sprint(err == nil),
			).Observe(time.Since(begin).Seconds())
			if err != nil {
				result = strings.Join([]string{result, err.Error()}, " ")
				if err != ErrNoop {
					inst.Log("err", err)
					result = "Failed: " + result
				}
			}
			if result != "" {
				updateJob(result)
				actions[i].Result = result
			}
			if err == ErrNoop {
				break // Release is a noop, let's get outta here!
			}
			if err != nil {
				return err // Something went wrong, abort the release
			}
		}
	}

	return nil
}

func CalculateUpdates(services []platform.Service, images instance.ImageMap, printf func(string, ...interface{})) map[flux.ServiceID][]ContainerUpdate {
	updateMap := map[flux.ServiceID][]ContainerUpdate{}
	for _, service := range services {
		containers, err := service.ContainersOrError()
		if err != nil {
			printf("service %s does not have images associated: %s", service.ID, err)
			continue
		}
		for _, container := range containers {
			currentImageID := flux.ParseImageID(container.Image)
			latestImage := images.LatestImage(currentImageID.Repository())
			if latestImage == nil {
				continue
			}

			if currentImageID == latestImage.ID {
				printf("Service %s image %s is already the latest one; skipping.", service.ID, currentImageID)
				continue
			}

			updateMap[service.ID] = append(updateMap[service.ID], ContainerUpdate{
				Container: container.Name,
				Current:   currentImageID,
				Target:    latestImage.ID,
			})
		}
	}
	return updateMap
}

// Release helpers.

type ContainerUpdate struct {
	Container string
	Current   flux.ImageID
	Target    flux.ImageID
}

// ReleaseAction Do funcs

func (r *Releaser) releaseActionPrintf(format string, args ...interface{}) ReleaseAction {
	return ReleaseAction{
		Name:        "printf",
		Description: fmt.Sprintf(format, args...),
		Do: func(_ *ReleaseContext) (res string, err error) {
			return "", nil
		},
	}
}

func (r *Releaser) releaseActionFetchPlatformServices(getServices ServiceSelector) ReleaseAction {
	return ReleaseAction{
		Name:        "fetch_platform_services",
		Description: fmt.Sprintf("Fetch %s from the platform", getServices),
		Do: func(rc *ReleaseContext) (res string, err error) {
			rc.Services, err = getServices.SelectServices(rc.Instance)
			if err != nil {
				return "", errors.Wrap(err, "fetching platform services")
			}
			if len(rc.Services) == 0 {
				return "No selected services found.", ErrNoop
			}
			return fmt.Sprintf("Found %d selected services.", len(rc.Services)), nil
		},
	}
}

func (r *Releaser) releaseActionClone() ReleaseAction {
	return ReleaseAction{
		Name:        "clone",
		Description: "Clone the config repo.",
		Do: func(rc *ReleaseContext) (res string, err error) {
			stderr, err := rc.CloneRepo()
			if err != nil {
				return stderr, errors.Wrap(err, "clone the config repo")
			}
			return fmt.Sprintf("Cloned %s", rc.RepoURL()), nil
		},
	}
}

func (r *Releaser) releaseActionFindDefinitions(services ServiceSelector) ReleaseAction {
	return ReleaseAction{
		Name:        "find_definitions",
		Description: fmt.Sprintf("Find definition files for %s", services),
		Do: func(rc *ReleaseContext) (res string, err error) {
			var skipped []flux.ServiceID
			rc.Definitions, skipped, err = services.SelectDefinitions(rc.RepoPath())
			if err != nil {
				return "", err
			}

			if len(rc.Definitions) <= 0 {
				return "No definition files found.", ErrNoop
			}

			var status []string
			if len(skipped) > 0 {
				for _, service := range skipped {
					status = append(status, fmt.Sprintf("No definition file found for %s; skipping", service))
				}
			}
			status = append(status, fmt.Sprintf("Found %d definition files.", len(rc.Definitions)))
			return strings.Join(status, "\n"), nil
		},
	}
}

func (r *Releaser) releaseActionCheckForNewImages(images ImageSelector) ReleaseAction {
	return ReleaseAction{
		Name:        "check_for_new_images",
		Description: fmt.Sprintf("Check registry for %s", images),
		Do: func(rc *ReleaseContext) (res string, err error) {
			// Fetch the image metadata
			rc.Images, err = images.SelectImages(rc.Instance, rc.Services)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Found %d new images", len(rc.Images)), nil
		},
	}
}

func (r *Releaser) releaseActionUpdateDefinitions(images ImageSelector) ReleaseAction {
	return ReleaseAction{
		Name:        "update_definition",
		Description: fmt.Sprintf("Update definition files to %s", images),
		Do: func(rc *ReleaseContext) (res string, err error) {
			updateMap := CalculateUpdates(rc.Services, rc.Images, func(format string, args ...interface{}) {})
			if len(updateMap) <= 0 {
				return "All selected services are running the requested images.", ErrNoop
			}

			definitionCount := 0
			for service, updates := range updateMap {
				// Update all definition files for this service. (should only be one)
				for path, definition := range rc.Definitions[service] {
					// We keep overwriting the same def, to handle multiple
					// images in a single file.
					updatedDefinition := definition
					for _, update := range updates {
						// Note 1: UpdateDefinition parses the target (new) image
						// name, extracts the repository, and only mutates the line(s)
						// in the definition that match it. So for the time being we
						// ignore the current image. UpdateDefinition could be
						// updated, if necessary.
						//
						// Note 2: we keep overwriting the same def, to handle multiple
						// images in a single file.
						updatedDefinition, err = kubernetes.UpdateDefinition(updatedDefinition, string(update.Target), ioutil.Discard)
						if err != nil {
							return "", errors.Wrapf(err, "updating definition for %s", service)
						}
					}
					if string(definition) != string(updatedDefinition) {
						if _, ok := rc.UpdatedDefinitions[service]; !ok {
							rc.UpdatedDefinitions[service] = map[string][]byte{}
						}
						rc.UpdatedDefinitions[service][path] = updatedDefinition
						definitionCount++
					}
				}
			}
			return fmt.Sprintf("Updated %d definition files for %d services", definitionCount, len(rc.UpdatedDefinitions)), nil
		},
	}
}

func (r *Releaser) releaseActionCommitAndPush(msg string) ReleaseAction {
	return ReleaseAction{
		Name:        "commit_and_push",
		Description: "Commit and push the config repo.",
		Do: func(rc *ReleaseContext) (res string, err error) {
			if len(rc.UpdatedDefinitions) == 0 {
				return "No definitions updated, nothing to commit", nil
			}

			path := rc.RepoPath()
			if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
				return "", fmt.Errorf("the repo path (%s) is not valid", path)
			}
			// Write each changed definition file back, so commit/push works.
			for service, definitions := range rc.UpdatedDefinitions {
				for path, definition := range definitions {
					fi, err := os.Stat(path)
					if err != nil {
						return "", errors.Wrapf(err, "writing new definition file for %s: %s", service, path)
					}

					if err := ioutil.WriteFile(path, definition, fi.Mode()); err != nil {
						return "", errors.Wrapf(err, "writing new definition file for %s: %s", service, path)
					}
				}
			}

			result, err := rc.CommitAndPush(msg)
			if err == nil && result == "" {
				return "Pushed commit: " + msg, nil
			}
			return result, err
		},
	}
}

func service2string(a []flux.ServiceID) []string {
	s := make([]string, len(a))
	for i := range a {
		s[i] = string(a[i])
	}
	return s
}

func (r *Releaser) releaseActionApplyToPlatform(msg string) ReleaseAction {
	return ReleaseAction{
		Name: "apply_to_platform",
		// TODO: take the platform here and *ask* it what type it is, instead of
		// assuming kubernetes.
		Description: "Rolling-update to Kubernetes",
		Do: func(rc *ReleaseContext) (res string, err error) {
			definitions := rc.UpdatedDefinitions
			if len(definitions) == 0 {
				definitions = rc.Definitions
			}

			cause := strconv.Quote(msg)

			// We'll collect results for each service release.
			results := map[flux.ServiceID]error{}

			// Collect definitions for each service release.
			var defs []platform.ServiceDefinition
			// If we're regrading our own image, we want to do that
			// last, and "asynchronously" (meaning we probably won't
			// see the reply).
			var asyncDefs []platform.ServiceDefinition

			for service, changedDefs := range definitions {
				namespace, serviceName := service.Components()
				// Apply each changed definition file for this service
				for _, def := range changedDefs {
					switch serviceName {
					case FluxServiceName, FluxDaemonName:
						rc.Instance.LogEvent(namespace, serviceName, "Starting "+cause+". (no result expected)")
						asyncDefs = append(asyncDefs, platform.ServiceDefinition{
							ServiceID:     service,
							NewDefinition: def,
						})
					default:
						rc.Instance.LogEvent(namespace, serviceName, "Starting "+cause)
						defs = append(defs, platform.ServiceDefinition{
							ServiceID:     service,
							NewDefinition: def,
						})
					}
				}
			}

			// Execute the releases as a single transaction.
			// Splat any errors into our results map.
			transactionErr := rc.Instance.PlatformApply(defs)
			if transactionErr != nil {
				switch err := transactionErr.(type) {
				case platform.ApplyError:
					for id, applyErr := range err {
						results[id] = applyErr
					}
				default: // assume everything failed, if there was a coverall error
					for service, _ := range definitions {
						results[service] = transactionErr
					}
				}
			}

			// Report individual service release results.
			for service := range definitions {
				namespace, serviceName := service.Components()
				switch serviceName {
				case FluxServiceName, FluxDaemonName:
					continue
				default:
					if err := results[service]; err == nil { // no entry = nil error
						rc.Instance.LogEvent(namespace, serviceName, msg+". done")
					} else {
						rc.Instance.LogEvent(namespace, serviceName, msg+". error: "+err.Error()+". failed")
					}
				}
			}

			// Lastly, services for which we don't expect a result
			// (i.e., ourselves). This will kick off the release in
			// the daemon, which will cause Kubernetes to restart the
			// service. In the meantime, however, we will have
			// finished recording what happened, as part of a graceful
			// shutdown. So the only thing that goes missing is the
			// result from this release call.
			if len(asyncDefs) > 0 {
				go func() {
					rc.Instance.PlatformApply(asyncDefs)
				}()
			}

			return "", transactionErr
		},
	}
}
