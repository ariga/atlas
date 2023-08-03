module ariga.io/atlas/internal/integration

go 1.20

replace ariga.io/atlas => ../../

replace ariga.io/atlas/cmd/atlas => ../../cmd/atlas

require (
	ariga.io/atlas v0.12.1-0.20230726093600-73399935e167
	ariga.io/atlas/cmd/atlas v0.12.2-0.20230803123152-64bd8681250f
	github.com/go-sql-driver/mysql v1.7.0
	github.com/hashicorp/hcl/v2 v2.13.0
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/pkg/diff v0.0.0-20210226163009-20ebb0f2a09e
	github.com/rogpeppe/go-internal v1.9.0
	github.com/stretchr/testify v1.8.2
	github.com/zclconf/go-cty v1.10.0
)

require (
	cloud.google.com/go v0.103.0 // indirect
	cloud.google.com/go/compute v1.7.0 // indirect
	cloud.google.com/go/iam v0.3.0 // indirect
	cloud.google.com/go/secretmanager v1.5.0 // indirect
	entgo.io/ent v0.12.4-0.20230726082433-91c7fcc68504 // indirect
	github.com/1lann/promptui v0.8.1-0.20220708222609-81fad96dd5e1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20220816024939-bc8df83d7b9d // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230512164433-5d1fd1a340c9 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/auxten/postgresql-parser v1.0.1 // indirect
	github.com/aws/aws-sdk-go v1.44.68 // indirect
	github.com/aws/aws-sdk-go-v2 v1.16.8 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.15.15 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.10 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.15.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.10 // indirect
	github.com/aws/smithy-go v1.12.0 // indirect
	github.com/benbjohnson/clock v1.1.0 // indirect
	github.com/certifi/gocertifi v0.0.0-20200922220541-2c3bb06c6054 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cockroachdb/apd v1.1.1-0.20181017181144-bced77f817b4 // indirect
	github.com/cockroachdb/errors v1.8.2 // indirect
	github.com/cockroachdb/logtags v0.0.0-20190617123548-eb05cc24525f // indirect
	github.com/cockroachdb/redact v1.0.8 // indirect
	github.com/cockroachdb/sentry-go v0.6.1-cockroachdb.2 // indirect
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/getsentry/raven-go v0.2.0 // indirect
	github.com/go-openapi/inflect v0.19.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/libsql/libsql-client-go v0.0.0-20230602133133-5905f0c4f8a5 // indirect
	github.com/libsql/sqlite-antlr4-parser v0.0.0-20230512205400-b2348f0d1196 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pingcap/errors v0.11.5-0.20210425183316-da1aaba5fb63 // indirect
	github.com/pingcap/log v0.0.0-20210625125904-98ed8e2eb1c7 // indirect
	github.com/pingcap/tidb/parser v0.0.0-20220817134052-9709249e523a // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v1.7.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/vektah/gqlparser/v2 v2.5.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	gocloud.dev v0.27.0 // indirect
	golang.org/x/exp v0.0.0-20220722155223-a9213eeb770e // indirect
	golang.org/x/mod v0.10.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/oauth2 v0.0.0-20220722155238-128564f6959c // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/tools v0.8.1-0.20230428195545-5283a0178901 // indirect
	golang.org/x/xerrors v0.0.0-20220609144429-65e65417b02f // indirect
	google.golang.org/api v0.91.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220802133213-ce4fa296bf78 // indirect
	google.golang.org/grpc v1.48.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)
