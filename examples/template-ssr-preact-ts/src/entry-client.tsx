import "./index.css"
import { hydrate } from "preact"
import { App } from "./app"
import { getClientSideProps } from "govite"

hydrate(
	<App {...getClientSideProps()} />,
	document.getElementById("app") as HTMLElement,
)
