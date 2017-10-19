package exoscale

import "gopkg.in/amz.v2/s3"

func ConvertAcl(acl string) s3.ACL {
    switch acl {
    case "private":
        return s3.Private
    case "public-read":
        return s3.PublicRead
    case "public-read-write":
        return s3.PublicReadWrite
    case "authenticated-read":
        return s3.AuthenticatedRead
    case "bucket-owner-read":
        return s3.BucketOwnerRead
    case "bucket-owner-full-control":
        return s3.BucketOwnerFull
    }

    /* if we don't know what it is, then it'll just be private */
    return s3.Private
}