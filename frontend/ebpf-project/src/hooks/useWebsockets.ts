import { useEffect, useRef, useState } from "react"
import type { WSMessage, FunctionEvent, CallEvent } from "../types/events"
import { discriminator } from "../lib/parser"
// by default it has used the input given in the UI
const DEFAULT_URL = "ws://192.168.110.128:8080/ws"

export function WebsocketConnectionEntrance(url: string, onReceivedMessage: (ws_message: WSMessage ) => any){
    const [status, setStatus] = useState<"connecting" | "open" | "closed" | "error">("connecting")
    const ws = useRef<WebSocket | null>(null)

    useEffect(() => {
        if(!url){
            url = DEFAULT_URL
        }
        let current_ws = ws.current = new WebSocket(url)

        current_ws.onopen = () => {
            setStatus("open")
        }
        // receiving the message open websocket
        current_ws.onmessage = (ws_message_event) =>{
            console.log(ws_message_event)
            const parsedMessage : WSMessage =  JSON.parse(ws_message_event.data)
            console.log(parsedMessage)
            // const actualMessage = discriminator(parsedMessage)
            // console.log(actualMessage)
            if (parsedMessage){
                 onReceivedMessage(parsedMessage)
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

