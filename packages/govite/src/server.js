import * as fs from "node:fs/promises"
import { createServer } from "vite"
import express from "express"
import { cwd } from "node:process"

// Constants
const port = Number(process.env["PORT"] || 6543)
const hmrPort = Number(process.env["HMR_PORT"] || 26543)
const base = process.env["BASE"] || "/"
const serverPort = Number(process.env["SERVER_PORT"])

// Create http server
const app = express()

const vite = await createServer({
	appType: "custom",
	base,
	root: cwd(),
	server: {
		middlewareMode: true,
		hmr: {
			port: hmrPort,
		},
		port: serverPort,
	},
})

app.use(vite.middlewares)

const htmlTemplate = await fs.readFile("./index.html", "utf-8")

// Serve HTML
app.use("*", async (req, res) => {
	console.log(
		"Request",
		req.url,
		req.headers.host,
		req.headers.origin,
		req.headers.referer,
		req.headers["user-agent"],
	)

	try {
		const url = new URL(req.url ?? "", `http://${req.headers.host}`)

		const propsParam = url.searchParams.get("props")
		const props = propsParam ? JSON.parse(decodeURIComponent(propsParam)) : {}
		const definesParam = url.searchParams.get("defines")
		const defines = definesParam
			? JSON.parse(decodeURIComponent(definesParam))
			: {}
		console.log("propsParam", propsParam, "definesParam", definesParam)

		const template = await vite.transformIndexHtml(
			url.pathname.replace(base, ""),
			htmlTemplate,
		)
		const { render } = await vite.ssrLoadModule("/src/entry-server")

		const rendered = await render(props)

		const html = template
			.replace("</head>", `${rendered.head ?? ""}</head>`)
			.replace(
				"</head>",
				rendered.css ? `<style>${rendered.css}</style>\n</head>` : "</head>",
			)
			.replace(
				"</head>",
				`<script>
          window.__INITIAL_STATE__ = ${JSON.stringify(props)};
          window.__DEFINES__ = ${JSON.stringify(defines)};
          window.__HMR_CONFIG_NAME__ = undefined;
          window.__BASE__ = "${base}";
          window.__SERVER_HOST__ = window.location.origin;
          window.__HMR_PROTOCOL__ = "ws";
          window.__HMR_PORT__ = "${hmrPort}";
          window.__HMR_HOSTNAME__ = "localhost";
          window.__HMR_BASE__ = "${base}";
          window.__HMR_DIRECT_TARGET__ = false;
          window.__HMR_ENABLE_OVERLAY__ = false;
          window.__HMR_TIMEOUT__ = 30;
        </script>
        </head>`,
			)
			.replace(
				'<div id="app"></div>',
				`<div id="app">${rendered.html ?? ""}</div>`,
			)

		res.setHeader("Content-Type", "text/html").status(200).end(html)
	} catch (e) {
		/** @type {Error} */
		// @ts-ignore
		const error = e

		vite.ssrFixStacktrace(error)

		console.log(error.stack)

		res.status(500).setHeader("Content-Type", "text/plain").end(error.stack)
	}
})

app.listen(port, "0.0.0.0")
