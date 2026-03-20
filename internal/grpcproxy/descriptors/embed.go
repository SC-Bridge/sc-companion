package descriptors

import _ "embed"

// DescriptorSet is the compiled FileDescriptorSet from all SC proto definitions.
// Generate with: protoc --descriptor_set_out=sc.pb --include_imports *.proto
//
//go:embed sc.pb
var DescriptorSet []byte
