pub mod callstack;
use std::{f64::consts::E, iter::Map, print, println};
use anyhow::bail;
#[cfg(test)]
mod tests {
    use std::default;

use super::*;
    #[test]
    fn call_stack_test(){
        let mut cs =callstack::CallStack::new();
        let k = callstack::UniqueKeyIdentifier{pid : 1, generic_key: 2};
        let root = cs.enter(k, "A".into(), 0);
        assert_eq!(root.caller, "");
        let edge = cs.enter(k, "B".into(), 10);
        assert_eq!((edge.caller.as_str(), edge.callee.as_str()), ("A", "B"));
        assert_eq!(edge.trace_id, root.trace_id);
        assert_eq!(cs.exit(k,30).unwrap().self_time,20);
        cs.enter(k, "C".into(), 15);
        assert_eq!(cs.exit(k,35).unwrap().self_time,20);
        let a = cs.exit(k,50).unwrap();
        assert_eq!((a.function.as_str(), a.self_time, a.total_time), ("A", 10, 50));

    }

    #[test]
    fn exit_without_enter_is_none(){
        let mut cs =callstack::CallStack::new();
        let k = callstack::UniqueKeyIdentifier{pid : 1, generic_key: 2};
        assert!(cs.exit(k, 10).is_none());
    }




#[derive(Debug)]
#[derive(PartialEq)]
pub enum EventType {
    Enter,
    Exit,
}

impl TryFrom<u8> for EventType {
    type Error = anyhow::Error;
    fn try_from(value: u8) -> Result<Self,Self::Error> {
        match value {
            0 => Ok(EventType::Enter),
            1 => Ok(EventType::Exit),
            _ =>  bail!("Invalid type")
   
        }
    }
}
#[derive(Debug)]
pub struct MockData {
    event_type: EventType,
    thread_id: u32,
    process_id: u32,
    cookie: u64,
    time_stamp: u64,
}

fn parse_bytes(bytes: &[u8]) -> MockData {
    let event_type: EventType = EventType::try_from(bytes[0]).unwrap();
    let thread_id = u32::from_le_bytes(bytes[1..5].try_into().unwrap());
    let process_id = u32::from_le_bytes(bytes[5..9].try_into().unwrap());
    let cookie = u64::from_le_bytes(bytes[9..17].try_into().unwrap());
    let time_stamp = u64::from_le_bytes(bytes[17..25].try_into().unwrap());
        MockData{
            event_type,
            thread_id,
            process_id,
            cookie,
            time_stamp,
        }
      
}
// 1731024000,18.5,82,3.4";
fn parse_text(mut string: &str) -> bool {
    let vector: Vec<&str>= string.split(",").collect();
    // let event_type: EventType = EventType::try_from(mock_parse());
    let thread_id: u32 = mock_parse(vector[1]);
    let process_id: u32 = mock_parse(vector[2]);
    let cookie: u64 = vector[3].parse::<u64>().unwrap();
    let time_stamp: u32 = mock_parse(vector[4]);

    true
}

fn mock_parse(string: &str) -> u32 {
    let mut total: u32 = 0;
    for (index, byte) in string.bytes().rev().enumerate() {
        let digit = (byte - b'0') as u32;
        total = digit * 10_u32.pow(index as u32) + total;
    }
    total
}

#[test]
fn test_a() {
    let bytes: [u8; 28] = [
        1, 43, 0, 0, 0, 0, 0, 0, 51, 0, 0, 0, 12, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0,
    ];
    let data : MockData= parse_bytes(&bytes);
    print!("{:?}" , data);
    assert_eq!(data.event_type, EventType::Exit)
}


}
