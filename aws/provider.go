package aws

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/manvalls/terraform-provider-wasabi/aws/internal/mutexkv"
)

// Provider returns a *schema.Provider.
func Provider() *schema.Provider {
	// TODO: Move the validation to this, requires conditional schemas
	// TODO: Move the configuration to this, requires validation

	// The actual provider
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["access_key"],
				DefaultFunc: schema.EnvDefaultFunc("WASABI_ACCESS_KEY", nil),
			},

			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: descriptions["secret_key"],
				DefaultFunc: schema.EnvDefaultFunc("WASABI_SECRET_KEY", nil),
			},

			"profile": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["profile"],
			},

			"assume_role": assumeRoleSchema(),

			"shared_credentials_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["shared_credentials_file"],
			},

			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: descriptions["token"],
			},

			"region": {
				Type:         schema.TypeString,
				Required:     true,
				DefaultFunc:  schema.EnvDefaultFunc("WASABI_REGION", nil),
				Description:  descriptions["region"],
				InputDefault: "us-east-1",
			},

			"max_retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     25,
				Description: descriptions["max_retries"],
			},

			"allowed_account_ids": {
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"forbidden_account_ids"},
				Set:           schema.HashString,
			},

			"forbidden_account_ids": {
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"allowed_account_ids"},
				Set:           schema.HashString,
			},

			"endpoints": endpointsSchema(),

			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: descriptions["insecure"],
			},

			"skip_credentials_validation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: descriptions["skip_credentials_validation"],
			},

			"skip_requesting_account_id": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: descriptions["skip_requesting_account_id"],
			},

			"skip_metadata_api_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: descriptions["skip_metadata_api_check"],
			},

			"s3_force_path_style": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: descriptions["s3_force_path_style"],
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"wasabi_account_alias":   dataSourceAwsIamAccountAlias(),
			"wasabi_group":           dataSourceAwsIAMGroup(),
			"wasabi_policy":          dataSourceAwsIAMPolicy(),
			"wasabi_policy_document": dataSourceAwsIamPolicyDocument(),
			"wasabi_role":            dataSourceAwsIAMRole(),
			"wasabi_user":            dataSourceAwsIAMUser(),
			"wasabi_bucket":          dataSourceAwsS3Bucket(),
			"wasabi_bucket_object":   dataSourceAwsS3BucketObject(),
			"wasabi_bucket_objects":  dataSourceAwsS3BucketObjects(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"wasabi_access_key":                 resourceAwsIamAccessKey(),
			"wasabi_account_alias":              resourceAwsIamAccountAlias(),
			"wasabi_account_password_policy":    resourceAwsIamAccountPasswordPolicy(),
			"wasabi_group_policy":               resourceAwsIamGroupPolicy(),
			"wasabi_group":                      resourceAwsIamGroup(),
			"wasabi_group_membership":           resourceAwsIamGroupMembership(),
			"wasabi_group_policy_attachment":    resourceAwsIamGroupPolicyAttachment(),
			"wasabi_policy":                     resourceAwsIamPolicy(),
			"wasabi_policy_attachment":          resourceAwsIamPolicyAttachment(),
			"wasabi_role_policy_attachment":     resourceAwsIamRolePolicyAttachment(),
			"wasabi_role_policy":                resourceAwsIamRolePolicy(),
			"wasabi_role":                       resourceAwsIamRole(),
			"wasabi_user_group_membership":      resourceAwsIamUserGroupMembership(),
			"wasabi_user_policy_attachment":     resourceAwsIamUserPolicyAttachment(),
			"wasabi_user_policy":                resourceAwsIamUserPolicy(),
			"wasabi_user":                       resourceAwsIamUser(),
			"wasabi_user_login_profile":         resourceAwsIamUserLoginProfile(),
			"wasabi_bucket":                     resourceAwsS3Bucket(),
			"wasabi_bucket_policy":              resourceAwsS3BucketPolicy(),
			"wasabi_bucket_public_access_block": resourceAwsS3BucketPublicAccessBlock(),
			"wasabi_bucket_object":              resourceAwsS3BucketObject(),
		},
	}

	provider.ConfigureFunc = func(d *schema.ResourceData) (interface{}, error) {
		terraformVersion := provider.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}
		return providerConfigure(d, terraformVersion)
	}

	return provider
}

var descriptions map[string]string
var endpointServiceNames []string

func init() {
	descriptions = map[string]string{
		"region": "The region where AWS operations will take place. Examples\n" +
			"are us-east-1, us-west-2, etc.",

		"access_key": "The access key for API operations. You can retrieve this\n" +
			"from the 'Security & Credentials' section of the AWS console.",

		"secret_key": "The secret key for API operations. You can retrieve this\n" +
			"from the 'Security & Credentials' section of the AWS console.",

		"profile": "The profile for API operations. If not set, the default profile\n" +
			"created with `aws configure` will be used.",

		"shared_credentials_file": "The path to the shared credentials file. If not set\n" +
			"this defaults to ~/.aws/credentials.",

		"token": "session token. A session token is only required if you are\n" +
			"using temporary security credentials.",

		"max_retries": "The maximum number of times an AWS API request is\n" +
			"being executed. If the API request still fails, an error is\n" +
			"thrown.",

		"endpoint": "Use this to override the default service endpoint URL",

		"insecure": "Explicitly allow the provider to perform \"insecure\" SSL requests. If omitted," +
			"default value is `false`",

		"skip_credentials_validation": "Skip the credentials validation via STS API. " +
			"Used for AWS API implementations that do not have STS available/implemented.",

		"skip_requesting_account_id": "Skip requesting the account ID. " +
			"Used for AWS API implementations that do not have IAM/STS API and/or metadata API.",

		"skip_medatadata_api_check": "Skip the AWS Metadata API check. " +
			"Used for AWS API implementations that do not have a metadata api endpoint.",

		"s3_force_path_style": "Set this to true to force the request to use path-style addressing,\n" +
			"i.e., http://s3.amazonaws.com/BUCKET/KEY. By default, the S3 client will\n" +
			"use virtual hosted bucket addressing when possible\n" +
			"(http://BUCKET.s3.amazonaws.com/KEY). Specific to the Amazon S3 service.",
	}

	endpointServiceNames = []string{
		"iam",
		"s3",
		"sts",
	}
}

func providerConfigure(d *schema.ResourceData, terraformVersion string) (interface{}, error) {
	region := d.Get("region").(string)

	config := Config{
		AccessKey:     d.Get("access_key").(string),
		SecretKey:     d.Get("secret_key").(string),
		Profile:       d.Get("profile").(string),
		Token:         d.Get("token").(string),
		Region:        region,
		CredsFilename: d.Get("shared_credentials_file").(string),
		Endpoints: map[string]string{
			"sts": "https://sts." + region + ".wasabisys.com",
			"iam": "https://iam." + region + ".wasabisys.com",
			"s3":  "https://s3." + region + ".wasabisys.com",
		},
		MaxRetries:              d.Get("max_retries").(int),
		Insecure:                d.Get("insecure").(bool),
		SkipCredsValidation:     d.Get("skip_credentials_validation").(bool),
		SkipRequestingAccountId: d.Get("skip_requesting_account_id").(bool),
		SkipMetadataApiCheck:    d.Get("skip_metadata_api_check").(bool),
		S3ForcePathStyle:        d.Get("s3_force_path_style").(bool),
		terraformVersion:        terraformVersion,
	}

	if l, ok := d.Get("assume_role").([]interface{}); ok && len(l) > 0 && l[0] != nil {
		m := l[0].(map[string]interface{})

		if v, ok := m["duration_seconds"].(int); ok && v != 0 {
			config.AssumeRoleDurationSeconds = v
		}

		if v, ok := m["external_id"].(string); ok && v != "" {
			config.AssumeRoleExternalID = v
		}

		if v, ok := m["policy"].(string); ok && v != "" {
			config.AssumeRolePolicy = v
		}

		if policyARNSet, ok := m["policy_arns"].(*schema.Set); ok && policyARNSet.Len() > 0 {
			for _, policyARNRaw := range policyARNSet.List() {
				policyARN, ok := policyARNRaw.(string)

				if !ok {
					continue
				}

				config.AssumeRolePolicyARNs = append(config.AssumeRolePolicyARNs, policyARN)
			}
		}

		if v, ok := m["role_arn"].(string); ok && v != "" {
			config.AssumeRoleARN = v
		}

		if v, ok := m["session_name"].(string); ok && v != "" {
			config.AssumeRoleSessionName = v
		}

		if transitiveTagKeySet, ok := m["transitive_tag_keys"].(*schema.Set); ok && transitiveTagKeySet.Len() > 0 {
			for _, transitiveTagKeyRaw := range transitiveTagKeySet.List() {
				transitiveTagKey, ok := transitiveTagKeyRaw.(string)

				if !ok {
					continue
				}

				config.AssumeRoleTransitiveTagKeys = append(config.AssumeRoleTransitiveTagKeys, transitiveTagKey)
			}
		}

		log.Printf("[INFO] assume_role configuration set: (ARN: %q, SessionID: %q, ExternalID: %q)", config.AssumeRoleARN, config.AssumeRoleSessionName, config.AssumeRoleExternalID)
	}

	endpointsSet := d.Get("endpoints").(*schema.Set)

	for _, endpointsSetI := range endpointsSet.List() {
		endpoints := endpointsSetI.(map[string]interface{})
		for _, endpointServiceName := range endpointServiceNames {
			config.Endpoints[endpointServiceName] = endpoints[endpointServiceName].(string)
		}
	}

	if v, ok := d.GetOk("allowed_account_ids"); ok {
		for _, accountIDRaw := range v.(*schema.Set).List() {
			config.AllowedAccountIds = append(config.AllowedAccountIds, accountIDRaw.(string))
		}
	}

	if v, ok := d.GetOk("forbidden_account_ids"); ok {
		for _, accountIDRaw := range v.(*schema.Set).List() {
			config.ForbiddenAccountIds = append(config.ForbiddenAccountIds, accountIDRaw.(string))
		}
	}

	return config.Client()
}

// This is a global MutexKV for use within this plugin.
var awsMutexKV = mutexkv.NewMutexKV()

func assumeRoleSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"duration_seconds": {
					Type:        schema.TypeInt,
					Optional:    true,
					Description: "Seconds to restrict the assume role session duration.",
				},
				"external_id": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Unique identifier that might be required for assuming a role in another account.",
				},
				"policy": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
				},
				"policy_arns": {
					Type:        schema.TypeSet,
					Optional:    true,
					Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"role_arn": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Amazon Resource Name of an IAM Role to assume prior to making API calls.",
				},
				"session_name": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Identifier for the assumed role session.",
				},
				"transitive_tag_keys": {
					Type:        schema.TypeSet,
					Optional:    true,
					Description: "Assume role session tag keys to pass to any subsequent sessions.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func endpointsSchema() *schema.Schema {
	endpointsAttributes := make(map[string]*schema.Schema)

	for _, endpointServiceName := range endpointServiceNames {
		endpointsAttributes[endpointServiceName] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			Description: descriptions["endpoint"],
		}
	}

	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: endpointsAttributes,
		},
	}
}
