import { Socket } from "node:net"
import { pid } from "node:process"

const socket = new Socket()

function log(msg, ...args) {
	console.log(`[runtime-v2] ${pid}: ${msg}`, ...args)
}

/** @param {Result} result */
function writeResult(result) {
	log("Sending result:", result)

	socket.write(JSON.stringify(result) + "\n")
}

log("Connecting to port:", process.env.PORT)

socket.connect(Number(process.env.PORT), () => {
	log("Connected to port:", process.env.PORT)

	socket.on("data", async (data) => {
		log("Received message:", data.toString("utf8"))

		const messages = data.toString("utf8").split("\n")

		messages.filter(Boolean).forEach(async (rawMessage) => {
			log("Processing message:", rawMessage)

			try {
				/** @type {Message} */
				const message = JSON.parse(rawMessage)

				if (message.type === "import") {
					try {
						const { default: content } = await import(message.content)

						writeResult({
							id: message.id,
							status: "success",
							content: await content,
						})
					} catch (error) {
						writeResult({
							id: message.id,
							status: "error",
							content:
								error instanceof Error ? error.message : JSON.stringify(error),
						})
					}
				} else if (message.type === "ping") {
					writeResult({
						id: message.id,
						status: "success",
						content: "pong",
					})
				} else {
					writeResult({
						id: message.id,
						status: "error",
						content: "Invalid message type",
					})
				}
			} catch (error) {
				writeResult({
					id: "INVALID_MESSAGE",
					status: "error",
					content: rawMessage,
				})
			}
		})
	})

	writeResult({
		id: "INITIALIZED",
		status: "success",
		content: "Initialized runtime-v2",
	})
})
