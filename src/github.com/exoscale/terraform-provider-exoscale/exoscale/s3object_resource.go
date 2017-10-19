package exoscale

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "os"

    "gopkg.in/amz.v2/s3"
    "github.com/hashicorp/terraform/helper/schema"
)

/**
 * Currently Key/Value pairs and CORS are not supported for bucket items.
 **/

 func s3ObjectResource() *schema.Resource {
    return &schema.Resource{
        Create: s3ObjectCreate,
        Read:   s3ObjectRead,
        Delete: s3ObjectDelete,
        Update: s3ObjectUpdate,

        Schema: map[string]*schema.Schema{
            "id": &schema.Schema{
                Type:       schema.TypeString,
                Computed:   true,
            },
            "bucket": &schema.Schema{
                Type:       schema.TypeString,
                ForceNew:   true,
                Required:   true,
            },
            "acl": &schema.Schema{
                Type:       schema.TypeString,
                ForceNew:   true,
                Optional:   true,
            },
            "key": &schema.Schema{
                Type:       schema.TypeString,
                ForceNew:   true,
                Required:   true,
            },
            "type": &schema.Schema{
                Type:       schema.TypeString,
                Required:   true,
            },
            "source": &schema.Schema{
                Type:       schema.TypeString,
                Optional:   true,
            },
            "content": &schema.Schema{
                Type:       schema.TypeString,
                Optional:   true,
            },
            "size": &schema.Schema{
                Type:       schema.TypeInt,
                Computed:   true,
            },
            "lastmodified": &schema.Schema{
                Type:       schema.TypeString,
                Computed:   true,
            },
        },
    }
 }

func s3ObjectCreate(d *schema.ResourceData, meta interface{}) error {
    session := GetS3Client(meta)
    bucket := session.Bucket(d.Get("bucket").(string))

    if d.Get("source") != "" && d.Get("content") != "" {
        return fmt.Errorf("Expect one of either source or content to be defined")
    }

    if d.Get("source") == "" && d.Get("content") == "" {
        return fmt.Errorf("One of source or content must be defined")
    }

    err := s3ObjectLoadData(d, bucket); if err != nil {
        return err
    }

    d.SetId(d.Get("bucket").(string) + "/" + d.Get("key").(string))

    return s3ObjectRead(d, meta)
}

func s3ObjectRead(d *schema.ResourceData, meta interface{}) error {
    session := GetS3Client(meta)
    bucket := session.Bucket(d.Get("bucket").(string))

    resp, err := bucket.List(d.Get("key").(string), "/", "", 100); if err != nil {
        return err
    }

    if len(resp.Contents) == 0 {
        return nil
    } else if len(resp.Contents) > 1 {
        return fmt.Errorf("Found too many keys with the name: %s\n", d.Get("key").(string))
    }

    d.Set("size", resp.Contents[0].Size)
    d.Set("lastmodified", resp.Contents[0].LastModified)

    return nil
}

func s3ObjectUpdate(d *schema.ResourceData, meta interface{}) error {
    /* Treate the update as a delete/create */
    session := GetS3Client(meta)
    bucket := session.Bucket(d.Get("bucket").(string))

    err := bucket.Del(d.Get("key").(string)); if err != nil {
        return err
    }

    err = s3ObjectLoadData(d, bucket); if err != nil {
        return err
    }

    return s3ObjectRead(d, meta)
}


func s3ObjectDelete(d *schema.ResourceData, meta interface{}) error {
    session := GetS3Client(meta)
    bucket := session.Bucket(d.Get("bucket").(string))

    err := bucket.Del(d.Get("key").(string))
    return err
}

func s3ObjectLoadData(d *schema.ResourceData, bucket *s3.Bucket) error {
    if d.Get("content") != "" {
        buffer := bytes.NewBufferString(d.Get("content").(string))
        err := bucket.Put(d.Get("key").(string), buffer.Bytes(),
            d.Get("type").(string), ConvertAcl(d.Get("acl").(string))); if err != nil {
            return err
        }
    } else {
        file, err := os.Open(d.Get("source").(string)); if err != nil {
            return err
        }

        fs, err := file.Stat(); if err != nil {
            return err
        }

        if fs.IsDir() {
            return fmt.Errorf("Specified location must be a file")
        }

        rf, err := ioutil.ReadFile(d.Get("source").(string)); if err != nil {
            return err
        }

        reader := bytes.NewReader(rf)
        length := fs.Size()

        err = bucket.PutReader(d.Get("key").(string), reader, length,
            d.Get("type").(string), ConvertAcl(d.Get("acl").(string))); if err != nil {
            return err
        }
    }

    return nil    
}