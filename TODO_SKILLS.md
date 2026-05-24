skill for Go ADK.

Custom Tools must follow funcArgs, funcResult, func pattern. funcArgs is a struct of the expected inputs. You have to do the Go json thing. funcResult is the result, duh. func is the func and it must have a signature like this: "func funcName(ctx tool.Context, args funcArgs) (funcResult, error)"

Use jsonschema to define and describe the arguments of the tools. Omitempty for non-required arguments or mutually exclusive arguments only.

You can do "dependency injection" to call tools that require a live client instance (like my Spotify client). Look at main.go for this.