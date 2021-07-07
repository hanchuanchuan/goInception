

### Install package


[goInception package](https://github.com/hanchuanchuan/goInception/releases)


### Source code 

- Golang v1.12 and above
- Go mod to manage package dependencies

```sh

# download source code
git clone https://github.com/hanchuanchuan/goInception

cd goInception

make parser

# build package
go build -o goInception tidb-server/main.go

```

#### start with configuration file

```sh
./goInception -config=config/config.toml
```


