package diff

// Types and procedures for diffing Kubernetes objects.
//
// The diffs so-generated are "semantic diffs", because certain fields
// may look different syntactically but are the same in effect; for
// instance, changing the order of environment entries.
