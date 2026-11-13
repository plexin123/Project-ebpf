use std::{format, path::Prefix};

use anyhow::{Context, bail};

#[derive(Debug)]
pub struct MockData {
    pub event_type: EventType,
    pub thread_id: u32,
    process_id: u32,
    cookie: u64,
    time_stamp: u64,
}

#[derive(Debug, PartialEq)]
pub enum EventType {
    Enter,
    Exit,
}

impl TryFrom<u8> for EventType {
    type Error = anyhow::Error;
    fn try_from(value: u8) -> Result<Self, Self::Error> {
        match value {
            0 => Ok(EventType::Enter),
            1 => Ok(EventType::Exit),
            _ => bail!("Invalid type"),
        }
    }
}

// class parser

fn parse_bytes(bytes: &[u8]) -> MockData {
    let event_type = EventType::try_from(bytes[0])
        .with_context(|| format!("it was not possible to parse the byte {}", bytes[0]))
        .unwrap();
    let thread_id = u32::from_le_bytes(bytes[1..5].try_into().unwrap());
    let process_id = u32::from_le_bytes(bytes[5..9].try_into().unwrap());
    let cookie = u64::from_le_bytes(bytes[9..17].try_into().unwrap());
    let time_stamp = u64::from_le_bytes(bytes[17..25].try_into().unwrap());
    MockData {
        event_type,
        thread_id,
        process_id,
        cookie,
        time_stamp,
    }
}



fn parse_text(text: &str) -> MockData {
    let vector: Vec<&str> = text.split(",").collect();
    // let event_type: EventType = EventType::try_from(mock_parse(vector[0]));
    let thread_id: u32 = number_parse(vector[1]) as u32;
    let process_id: u32 = number_parse(vector[2]) as u32;
    let cookie: u64 = number_parse(vector[3]);
    let time_stamp: u64 = number_parse(vector[4]);

    MockData {
        event_type: EventType::Enter,
        thread_id: thread_id,
        process_id: process_id,
        cookie: cookie,
        time_stamp: time_stamp,
    }
}
fn number_parse(string: &str) -> u64 {
    let mut total: u64 = 0;
    for (index, byte) in string.bytes().rev().enumerate() {
        let digit = (byte - b'0') as u64;
        total = digit * 10_u64.pow(index as u32) + total;
    }
    total
}
#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn test_parsing_bytes() {
        let bytes: [u8; 28] = [
            1, 43, 0, 0, 0, 176, 1, 0, 0, 0, 0, 0, 12, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0,
        ];
        let data: MockData = parse_bytes(&bytes);
        print!("{:?}", data);
        assert_eq!(data.event_type, EventType::Exit)
    }
    #[test]

    fn test_parsing_text() {
        let text = "1731024000,185,82,34,1123";
        let mock_data = parse_text(text);
        print!("{:?}", mock_data);
        assert_eq!(mock_data.process_id, 82);
    }
}
