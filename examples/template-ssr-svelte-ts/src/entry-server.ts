import App from "./App.svelte"

export function render() {
	// @ts-ignore
	const { html, head, css } = App.render()

	return {
		html,
		head,
		css: css.code,
	}
}
