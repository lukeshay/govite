export function getClientSideProps() {
	const props = window.__INITIAL_STATE__

	if (!props) {
		console.warn("No initial state found. Was this page rendered by govite?")
	}

	return props ?? {}
}
