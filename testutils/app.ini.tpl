APP_NAME = Gitea: Git with a cup of tea
RUN_USER = {{ .User }}
WORK_PATH =  {{ .Options.Workdir }}/workdir
RUN_MODE = prod

[database]
DB_TYPE = sqlite3
HOST = 127.0.0.1:3306
NAME = gitea
USER = gitea
PASSWD = 
SCHEMA = 
SSL_MODE = disable
PATH = {{ .Options.Workdir }}/data/gitea.db
LOG_SQL = false

[repository]
ROOT = {{ .Options.Workdir }}/data/gitea-repositories

[server]
SSH_DOMAIN = {{ .Options.BindIp }}
DOMAIN = {{ .Options.BindIp }}
HTTP_PORT = 3000
ROOT_URL = http://{{ .Options.BindIp }}:{{ .Options.BindPort }}/
APP_DATA_PATH = {{ .Options.Workdir }}/data
DISABLE_SSH = false
START_SSH_SERVER = true
SSH_PORT = 2222
LFS_START_SERVER = true
LFS_JWT_SECRET = fTBj80dl01088Ms-jKzlRuV1U8IuopTmgly6k7WYQDw
OFFLINE_MODE = true

[lfs]
PATH = {{ .Options.Workdir }}/data/lfs

[mailer]
ENABLED = false

[service]
REGISTER_EMAIL_CONFIRM = false
ENABLE_NOTIFY_MAIL = false
DISABLE_REGISTRATION = false
ALLOW_ONLY_EXTERNAL_REGISTRATION = false
ENABLE_CAPTCHA = false
REQUIRE_SIGNIN_VIEW = false
DEFAULT_KEEP_EMAIL_PRIVATE = false
DEFAULT_ALLOW_CREATE_ORGANIZATION = true
DEFAULT_ENABLE_TIMETRACKING = true
NO_REPLY_ADDRESS = noreply.localhost

[openid]
ENABLE_OPENID_SIGNIN = false
ENABLE_OPENID_SIGNUP = false

[cron.update_checker]
ENABLED = false

[session]
PROVIDER = file

[log]
MODE = console
LEVEL = info
ROOT_PATH = {{ .Options.Workdir }}/log

[repository.pull-request]
DEFAULT_MERGE_STYLE = merge

[repository.signing]
DEFAULT_TRUST_MODEL = committer

[security]
INSTALL_LOCK = true
INTERNAL_TOKEN = eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYmYiOjE3NDMwOTg2MDV9.XhT0aPyhek5OdQgkdNGp-Q9LD6di8qJhltFft3DRSQQ
PASSWORD_HASH_ALGO = pbkdf2

[oauth2]
JWT_SECRET = aHbcoExi1lTXbe4YCnnw-j_vXFcCepesd6HoTxrtPcY
