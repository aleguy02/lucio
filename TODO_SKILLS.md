skill for Go ADK.

Custom Tools must follow funcArgs, funcResult, func pattern. funcArgs is a struct of the expected inputs. Do the Go json thing. funcResult is the result, duh. func is the func and it must have a signature like this: "func funcName(ctx tool.Context, args funcArgs) (funcResult, error)"