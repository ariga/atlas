module ariga.io/atlas/cmd/atlas

go 1.22

replace ariga.io/atlas => ../..

require (
	ariga.io/atlas v0.19.1-0.20240203083654-5948b60a8e43
	entgo.io/ent v0.13.0
	github.com/1lann/promptui v0.8.1-0.20220708222609-81fad96dd5e1
	github.com/antlr4-go/antlr/v4 v4.13.0
	github.com/auxten/postgresql-parser v1.0.1
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/feature/rds/auth v1.2.16
	github.com/fatih/color v1.13.0
	github.com/go-sql-driver/mysql v1.7.1
	github.com/google/uuid v1.4.0
	github.com/hashicorp/hcl/v2 v2.13.0
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pganalyze/pg_query_go/v4 v4.2.3
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.2
	github.com/tursodatabase/libsql-client-go v0.0.0-20240220085343-4ae0eb9d0898
	github.com/vektah/gqlparser/v2 v2.5.1
	github.com/zclconf/go-cty v1.8.0
	gocloud.dev v0.36.0
	golang.org/x/exp v0.0.0-20230515195305-f3d0a9c9a5cc
	golang.org/x/mod v0.15.0
	golang.org/x/oauth2 v0.14.0
	google.golang.org/api v0.151.0
	google.golang.org/grpc v1.59.0 // indirect
)

require (
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v1.1.5 // indirect
	cloud.google.com/go/longrunning v0.5.4 // indirect
	cloud.google.com/go/secretmanager v1.11.4 // indirect
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230512164433-5d1fd1a340c9 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/aws/aws-sdk-go v1.49.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.24.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.12 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.25.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.44.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5 // indirect
	github.com/aws/smithy-go v1.19.0 // indirect
	github.com/certifi/gocertifi v0.0.0-20200922220541-2c3bb06c6054 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cockroachdb/apd v1.1.1-0.20181017181144-bced77f817b4 // indirect
	github.com/cockroachdb/errors v1.8.2 // indirect
	github.com/cockroachdb/logtags v0.0.0-20190617123548-eb05cc24525f // indirect
	github.com/cockroachdb/redact v1.0.8 // indirect
	github.com/cockroachdb/sentry-go v0.6.1-cockroachdb.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/getsentry/raven-go v0.2.0 // indirect
	github.com/go-openapi/inflect v0.19.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/libsql/sqlite-antlr4-parser v0.0.0-20230802215326-5cb5bb604475 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/pingcap/errors v0.11.5-0.20210425183316-da1aaba5fb63 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.6.1 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.4.0 // indirect
	golang.org/x/tools v0.17.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231120223509-83a465c0220f // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)
