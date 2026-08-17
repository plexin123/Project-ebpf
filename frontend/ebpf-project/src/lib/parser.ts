
// handle  json_object = Record<string, any>
function parser(json_object: Record<string, any>):Record<string, any> {
    var new_json_object : Record<string,any> = {}
    for(const name of Object.keys(json_object)){
        new_json_object[name.toLowerCase()] = json_object[name]
    }
    return new_json_object
}

function discriminator(json_object:Record<string,any>): WSMessage|CallEvent | FunctionEvent | null {
    const type_name : string = json_object["type"]
    const payload : any = json_object["payload"]
    switch(type_name){
        case "connection":
            return payload as CallEvent
        case "event":
            return payload as FunctionEvent
        default:
            console.warn(`Not defined type available ${type_name}`)
            return null
    }
}