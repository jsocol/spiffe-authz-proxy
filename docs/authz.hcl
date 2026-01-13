spiffeid "spiffe://example.org/workload/workload1" {
  path "/foo/*/bar/*" {
    methods = ["GET", "POST", "PATCH", "DELETE"]
  }

  path "/foo" {
    methods = ["GET"]
  }

  path "/bar/**" {
    methods = ["GET", "POST"]
  }

  path "/**" {
    methods = ["OPTIONS"]
  }
}

# TODO: Trust entire trustdomains?
# trustdomain "spiffe://example.org/" {
#   path "/foo/*" {
#     methods = ["GET"]
#   }
# }
