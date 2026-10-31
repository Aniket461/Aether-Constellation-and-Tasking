module aether/tasking

go 1.27.3

require (
	github.com/golang-migrate/migrate/v4 v4.20.1
	github.com/jackc/pgx/v5 v5.12.0
	aether/constellation v0.0.0
)

require (
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.3 // indirect
	github.com/joshuaferrara/go-satellite v0.0.0-20220611180459-512638c64e5b // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
)

replace aether/constellation => ../../constellation
