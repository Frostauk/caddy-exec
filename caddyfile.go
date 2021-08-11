package command

import (
	"encoding/json"
	"net/http"

	// Frostauk

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterGlobalOption("exec", parseGlobalCaddyfileBlock)
	httpcaddyfile.RegisterHandlerDirective("exec", parseHandlerCaddyfileBlock)
}

func newCommandFromDispenser(d *caddyfile.Dispenser) (cmd Cmd, err error) {
	cmd.UnmarshalCaddyfile(d)
	return
}

// parseHandlerCaddyfileBlock configures the handler directive from Caddyfile.
// Syntax:
//
//   exec [<matcher>] [<command> [<args...>]] {
//       command     <text>
//       args        <text>...
//       directory   <text>
//       timeout     <duration>
//       log         <log output module>
//       err_log     <log output module>
//       foreground
//       startup
//       shutdown
//   }
//
func parseHandlerCaddyfileBlock(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	cmd, err := newCommandFromDispenser(h.Dispenser)
	return Middleware{Cmd: cmd}, err
}

// parseGlobalCaddyfileBlock configures the "exec" global option from Caddyfile.
// Syntax:
//
//   exec [<command> [<args...>]] {
//       command     <text>...
//       args        <text>...
//       directory   <text>
//       timeout     <duration>
//       log         <log output module>
//       err_log     <log output module>
//       foreground
//       startup
//       shutdown
//   }
//
func parseGlobalCaddyfileBlock(d *caddyfile.Dispenser, prev interface{}) (interface{}, error) {
	var exec App

	// decode the existing value and merge to it.
	if prev != nil {
		if app, ok := prev.(httpcaddyfile.App); ok {
			if err := json.Unmarshal(app.Value, &exec); err != nil {
				return nil, d.Errf("internal error: %v", err)
			}
		}
	}

	cmd, err := newCommandFromDispenser(d)
	if err != nil {
		return nil, err
	}

	// global block commands are not necessarily bound to a route,
	// should default to running at startup.
	if len(cmd.At) == 0 {
		cmd.At = append(cmd.At, "startup")
	}

	// append command to global exec app.
	exec.Commands = append(exec.Commands, cmd)

	// tell Caddyfile adapter that this is the JSON for an app
	return httpcaddyfile.App{
		Name:  "exec",
		Value: caddyconfig.JSON(exec, nil),
	}, nil
}

// UnmarshalCaddyfile configures the handler directive from Caddyfile.
// Syntax:
//
//   exec [<matcher>] [<command> [<args...>]] {
//       command     <text>
//       args        <text>...
//       directory   <text>
//       timeout     <duration>
//       log         <log output module>
//       err_log     <log output module>
//       foreground
//       startup
//       shutdown
//   }
//
func (c *Cmd) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	// consume "exec", then grab the command, if present.
	if d.NextArg() && d.NextArg() {
		c.Command = d.Val()
	}

	// everything else are args, if present.
	c.Args = d.RemainingArgs()

	// Frostauk
	// c.Args = insert_placeholders(c, c.Args)

	// parse the next block
	return c.unmarshalBlock(d)
}

func (c *Cmd) unmarshalBlock(d *caddyfile.Dispenser) error {
	for d.NextBlock(0) {
		switch d.Val() {
		case "command":
			if c.Command != "" {
				return d.Err("command specified twice")
			}
			if !d.Args(&c.Command) {
				return d.ArgErr()
			}
			c.Args = d.RemainingArgs()
			// Frostauk
			// c.Args = insert_placeholders(c, c.Args)
		case "args":
			if len(c.Args) > 0 {
				return d.Err("args specified twice")
			}
			c.Args = d.RemainingArgs()
			// Frostauk
			// c.Args = insert_placeholders(c, c.Args)
		case "directory":
			if !d.Args(&c.Directory) {
				return d.ArgErr()
			}
		case "foreground":
			c.Foreground = true
		case "startup":
			c.At = append(c.At, "startup")
		case "shutdown":
			c.At = append(c.At, "shutdown")
		case "timeout":
			if !d.Args(&c.Timeout) {
				return d.ArgErr()
			}
		case "log":
			rawMessage, err := c.unmarshalLog(d)
			if err != nil {
				return err
			}
			c.StdWriterRaw = rawMessage
		case "err_log":
			rawMessage, err := c.unmarshalLog(d)
			if err != nil {
				return err
			}
			c.ErrWriterRaw = rawMessage
		default:
			return d.Errf("'%s' not expected", d.Val())
		}
	}

	return nil
}

func (c *Cmd) unmarshalLog(d *caddyfile.Dispenser) (json.RawMessage, error) {
	if !d.NextArg() {
		return nil, d.ArgErr()
	}
	moduleName := d.Val()

	// copied from caddy's source
	// TODO: raise the topic of log re-use by non-standard modules.
	var wo caddy.WriterOpener
	switch moduleName {
	case "stdout":
		wo = caddy.StdoutWriter{}
	case "stderr":
		wo = caddy.StderrWriter{}
	case "discard":
		wo = caddy.DiscardWriter{}
	default:
		modID := "caddy.logging.writers." + moduleName
		unm, err := caddyfile.UnmarshalModule(d, modID)
		if err != nil {
			return nil, err
		}
		var ok bool
		wo, ok = unm.(caddy.WriterOpener)
		if !ok {
			return nil, d.Errf("module %s (%T) is not a WriterOpener", modID, unm)
		}
	}
	return caddyconfig.JSONModuleObject(wo, "output", moduleName, nil), nil
}

func insert_placeholders(request *http.Request, a []string) []string {
	// Frostauk - Attempt to replace placeholders using Replacer.ReplaceKnown(input, empty string)
	// empty string (taken from ReplaceAll description): "Values that are empty string will be substituted with empty."

	// Cannot be done due to dynamic array size creation
	//var return_array = [len(a)]string{}

	// Taken from: https://blog.golang.org/slices-intro
	var return_array = make([]string, len(a))

	// Taken from: https://github.com/amalto/caddy-vars-regex/blob/5684763f4d6994e618863e11b6b86ff87671900a/varsregex.go#L28
	// var r *caddy.Replacer = caddy.Replacer.NewReplacer()

	// Initializes, but doesn't replace due to not having a ReplacerKey
	// var r *caddy.Replacer = caddy.NewReplacer()

	// Taken from: https://github.com/amalto/caddy-vars-regex/blob/5684763f4d6994e618863e11b6b86ff87671900a/varsregex.go#L76
	// var r *caddy.Replacer = req.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	// Won't work due to NewContext requiring an existing Context to create a new one.
	// var c *caddy.Context = caddy.NewContext(caddy.ReplacerCtxKey);
	// var r *caddy.Replacer = c.(*caddy.Replacer)

	// Doesn't work due to Cmd with context causing a crash.
	// var r *caddy.Replacer = c.context.Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	var r *caddy.Replacer = request.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	for i := range a {
		return_array[i] = r.ReplaceKnown(a[i], "")
		// return_array[i] = "TEST"
	}

	// TODO: Cancel, according to Context's description: https://pkg.go.dev/github.com/caddyserver/caddy/v2#Context

	return return_array
}
