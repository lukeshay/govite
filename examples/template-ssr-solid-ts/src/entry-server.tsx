import { renderToString } from "solid-js/web"
import App from "./App"

export function render(props: any) {
	const html = renderToString(() => <App time={String(props.time)} />)
	return { html }
}
