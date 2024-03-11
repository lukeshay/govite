import { createSignal } from "solid-js"
import "./App.css"

function App({ time }: { time: string }) {
	const [count, setCount] = createSignal(0)

	return (
		<div class="App">
			<div>
				<a href="https://vitejs.dev" target="_blank">
					<img src="/vite.svg" class="logo" alt="Vite logo" />
				</a>
				<a href="https://www.solidjs.com" target="_blank">
					<img src="/solid.svg" class="logo solid" alt="Solid logo" />
				</a>
			</div>
			<h1>Vite + Solid</h1>
			<div class="card">
				<button onClick={() => setCount((count) => count + 1)}>
					count is {count()}
				</button>
				<p>
					Edit <code>src/App.tsx</code> and save to test HMR
				</p>
			</div>
			<p class="read-the-docs">
				Click on the Vite and Solid logos to learn more
			</p>
			<p>Server time: {time}</p>
		</div>
	)
}

export default App
