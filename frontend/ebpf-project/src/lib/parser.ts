import type { CallEvent, FunctionEvent } from '../types/events'

export function discriminator(json_object: Record<string, any>): CallEvent | FunctionEvent | null {
    const type_name: string = json_object["name"]
    const payload: any = json_object["payload"]
    switch (type_name) {
        case "connection":
            return payload as CallEvent
        case "event":
            return payload as FunctionEvent
        default:
            console.warn(`Not defined type available ${type_name}`)
            return null
    }
}
