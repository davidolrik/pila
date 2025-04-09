# Pila™ - Stack all the things!

> [!WARNING]
> This is a work in progress, everything might change

## Usage

> [!NOTE]
> Only `multi-merge` have been implemented for now

```sh
# Show branch stack
$ pila ls

# Push branches to origin
$ pila publish --stack

# Push branches to origin, and create PRs if they don't exist
$ pila propose --stack

# Create new target branch, and take merge in named branches
$ pila multi-merge create -T [target] -B <branch a> -B <branch b>

# Create new target branch, and take merge in PRs with the given labels
$ pila multi-merge create -T [target] -L <label> -L <label b>
```
