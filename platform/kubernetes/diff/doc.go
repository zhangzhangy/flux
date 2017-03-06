package diff

// Procedures for diffing Kubernetes objects. At the most general,
// these operate on in memory data structures. To get other varieties,
// files can be read into those structures, or config exported from a
// running cluster.
//
// The diffs are so-called "semantic diffs", because certain fields
// may look different syntactically but are the same in effect; for
// instance, changing the order of environment entries.
