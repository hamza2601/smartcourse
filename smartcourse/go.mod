// Deprecated: this root module is no longer built. Each service is its own module
// under services/ and shared code lives in libs/common.
module smartcourse

go 1.26.0

replace smartcourse/libs/common => ./libs/common
