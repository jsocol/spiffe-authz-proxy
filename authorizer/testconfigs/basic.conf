spiffeid "spiffe://example.org/a/workload" {
  path "/foo/bar" {
    methods = ["GET"]
  }
}

spiffeid "spiffe://example.org/b/other/app" {
  path "/foo" {
    methods = ["GET", "POST"]
  }

  path "/foo/*" {
     methods = ["GET", "PATCH", "PUT", "DELETE"]
   }
}
