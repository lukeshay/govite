export type RenderResult = {
	html: string
	css?: string
	head?: string
}

declare global {
	interface Window {
		__INITIAL_STATE__: any
	}
}

export type RenderHandler = (props: any) => Promise<RenderResult> | RenderResult

/**
 * Retrieves the client side props from the window object.
 *
 * @returns {any} The client side props.
 */
export function getClientSideProps(): any
