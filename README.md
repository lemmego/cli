# Lemmego CLI

Lemmego CLI is used for generating the framework components.

## Installation

### For Linux & macOS Users

  ```sh
curl -fsSL https://raw.githubusercontent.com/lemmego/cli/refs/heads/main/installer.sh | sudo sh
  ```


## Usage

### Create a new project:

`lemmego new <your-module-name>`

> A new project will be created in your current directory (must be an empty dir)

### Generate a handlers file:

`lemmego g handlers post`

> A post_handlers.go file will be generated in your project under the ./internal/handlers directory.

### Generate a model file:

`lemmego g model post`

> A post.go file will be generated in your project under the ./internal/models directory.

### Generate a form file:

`lemmego g form post`

> A Form.tsx file will be generated in your project under the ./resources/js/Pages/Forms directory.

### Generate a HTTP request input file:

`lemmego g input post`

> A post_input.go file will be generated in your project under the ./internal/inputs directory.

### Generate a migration file:

`lemmego g migration create_users_table`

> A <timestamp>_create_users_table.go file will be generated in your project under the ./internal/migrations directory (if you haven't overridden the default MIGRATIONS_DIR env value).

All these commands also take an interactive flag (`-i`), where additional configuration option is provided:

```
lemmego g -i handlers
lemmego g -i model
lemmego g -i form
lemmego g -i input
lemmego g -i migration
```

## Contributing

Pull requests are welcome. For major changes, please open an issue first
to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

[MIT](https://choosealicense.com/licenses/mit/)