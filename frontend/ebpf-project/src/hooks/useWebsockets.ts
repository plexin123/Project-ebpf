import { useEffect, useRef, useState } from "react"
import type { WSMessage, FunctionEvent, CallEvent } from "../types/events"
import { discriminator } from "../lib/parser"
// by default it has used the input given in the UI
const DEFAULT_URL = "http://192.168.110.128:8080/ws"

export function WebsocketConnectionEntrance(url: string, onReceivedMessage: (ws_message : WSMessage | FunctionEvent |CallEvent | null ) => any){
    const [status, setStatus] = useState<"connecting" | "open" | "closed" | "error">("connecting")
    const ws = useRef<WebSocket | null>(null)

    useEffect(() => {
        let current_ws = ws.current = new WebSocket(url)

        current_ws.onopen = () => {
            setStatus("open")
        }

        current_ws.onmessage = (ws_message) =>{
            const parsedMessage =  JSON.parse(ws_message.data)
            const actualMessage = discriminator(parsedMessage)
            if (actualMessage){
                 onReceivedMessage(actualMessage)
            }
        
        }
        current_ws.onerror = () => {
            setStatus("error")
        }
        current_ws.onclose = () => {
            setStatus("closed")
            }

        return () =>{
            current_ws.close()
        }
    }, [url])

    return status
}

