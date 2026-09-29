

export function ConvertIntoMilliSeconds(nanoseconds: number): string {
    let seconds =  `${(nanoseconds / 1000000).toFixed(2)}ms` 
    return seconds
}