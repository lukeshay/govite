interface BaseMessage<Type extends string> {
	id: string
	port: number
	type: Type
	content: string
}

interface ImportMessage extends BaseMessage<"import"> {}
interface PingMessage extends BaseMessage<"ping"> {}

type Message = ImportMessage | PingMessage

interface Result {
	id: string
	status: "success" | "error"
	content: string
}
