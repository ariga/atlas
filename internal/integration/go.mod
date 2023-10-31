module ariga.io/atlas/internal/integration

go 1.21

replace ariga.io/atlas => ../../

replace ariga.io/atlas/cmd/atlas => ../../cmd/atlas

require (
	ariga.io/atlas v0.13.2-0.20230813181001-1fef462e7670
	ariga.io/atlas/cmd/atlas v0.13.2-0.20230813181001-1fef462e7670
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
	cloud.google.com/go v0.110.4 // indirect
	cloud.google.com/go/compute v1.21.0 // indirect
	cloud.google.com/go/iam v1.1.1 // indirect
	cloud.google.com/go/secretmanager v1.11.1 // indirect
	entgo.io/ent v0.12.4-0.20230726082433-91c7fcc68504 // indirect
	github.com/1lann/promptui v0.8.1-0.20220708222609-81fad96dd5e1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20220816024939-bc8df83d7b9d // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230512164433-5d1fd1a340c9 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/aws/aws-sdk-go v1.44.68 // indirect
	github.com/aws/aws-sdk-go-v2 v1.20.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.15.15 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.10 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.9 // indirect
	github.com/aws/aws-sdk-go-v2/feature/rds/auth v1.2.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.15.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.10 // indirect
	github.com/aws/smithy-go v1.14.1 // indirect
	github.com/benbjohnson/clock v1.1.0 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/go-openapi/inflect v0.19.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/googleapis/gax-go/v2 v2.11.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/libsql/libsql-client-go v0.0.0-20230602133133-5905f0c4f8a5 // indirect
	github.com/libsql/sqlite-antlr4-parser v0.0.0-20230512205400-b2348f0d1196 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/pganalyze/pg_query_go/v4 v4.2.3 // indirect
	github.com/pingcap/errors v0.11.5-0.20210425183316-da1aaba5fb63 // indirect
	github.com/pingcap/log v0.0.0-20210625125904-98ed8e2eb1c7 // indirect
	github.com/pingcap/tidb/parser v0.0.0-20220817134052-9709249e523a // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
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
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/oauth2 v0.10.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/tools v0.8.1-0.20230428195545-5283a0178901 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/api v0.126.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/grpc v1.58.3 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)
