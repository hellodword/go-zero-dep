# go-zero-dep

## Usage

```sh
git clone --depth=1 -b v4.17.0 --recurse-submodules https://github.com/jackc/pgx

(cd pgx && go mod tidy && go mod vendor)

go-zero-dep -src ./pgx -dst ./pgx-zero -mod "github.com/github/pgx-zero-dep"
```

## TODO

- [ ] sub folder of target module
- [ ] support go embed/generate
