import { createServer } from "vite"
import { fileURLToPath, URL } from "node:url"

const __dirname = fileURLToPath(new URL(".", import.meta.url))

const vite = await createServer({
	server: { middlewareMode: false },
	appType: "custom",
	root: __dirname,
})

export async function render(serverEntry, indexHtml, url, props) {
	const template = await vite.transformIndexHtml(url, indexHtml)
	const { render: viteRender } = await vite.ssrLoadModule(serverEntry, {
		fixStackTrace: true,
	})

	return viteRender(props)
}

export default await render(
	"/src/entry-server.tsx",
	`<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vite + Preact + TS</title>
  <script>
  window.__INITIAL_STATE__ = %s
</script>
</head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/entry-client.tsx"></script>
  </body>
</html>
`,
	"/",
	{ path: "/", time: "2024-03-08T16:37:57-06:00" },
)
