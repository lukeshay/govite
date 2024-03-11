import renderToString from "preact-render-to-string"
import { App } from "./app"
import { RenderHandler } from "@govite/govite"

export const render: RenderHandler = (props) => {
	const html = renderToString(<App {...props} />)
	return { html }
}
