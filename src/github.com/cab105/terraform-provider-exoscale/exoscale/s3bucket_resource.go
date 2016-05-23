package exoscale

import (
	"github.com/hashicorp/terraform/helper/schema"
)

/**
 * For buckets, we'll expect at least a name and acls.
 * exoscale also supports CORS, but unfortunately the
 * canonical go libraries do not support it at this time.
 **/

 func s3BucketResource() *schema.Resource {
 	return &schema.Resource{
 		Create: s3BucketCreate,
 		Read:	s3BucketRead,
 		Delete:	s3BucketDelete,

 		Schema: map[string]*schema.Schema{
 			"id": &schema.Schema{
 				Type:		schema.TypeString,
 				Computed:	true,
 			},
 			"bucket": &schema.Schema{
 				Type:		schema.TypeString,
 				ForceNew:	true,
 				Required:	true,
 			},
 			"acl": &schema.Schema{
 				Type:		schema.TypeString,
 				ForceNew:	true,
 				Optional:   true,
 			},
 		},
 	}
 }

 func s3BucketCreate(d *schema.ResourceData, meta interface{}) error {
    session := GetS3Client(meta)

    bucket := session.Bucket(d.Get("bucket").(string))
    acl := ConvertAcl(d.Get("acl").(string))

    err := bucket.PutBucket(acl); if err != nil {
        return err
    }

    d.SetId(d.Get("bucket").(string))

    return s3BucketRead(d, meta)
 }

 func s3BucketRead(d *schema.ResourceData, meta interface{}) error {
    session := GetS3Client(meta)

    bucket := session.Bucket(d.Id())
    d.Set("bucket", bucket.Name)
    /* Until the amz library supports reading CORS/ACLs return just what we have */

    return nil    
}

func s3BucketDelete(d *schema.ResourceData, meta interface{}) error {
    /* This will fail unless we delete all underlying items */
    session := GetS3Client(meta)
    bucket := session.Bucket(d.Id())

    err := bucket.DelBucket(); if err != nil {
        return err
    }

    return nil
}