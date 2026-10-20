
use std::collections::HashMap;
use uuid::Uuid;

#[derive(Debug,Clone,Copy,Hash,PartialEq,Eq)]
pub struct UniqueKeyIdentifier{
    pub pid : u32,
    pub generic_key : u32,
}

pub struct CallStack{
	map_trace_id   :  HashMap<UniqueKeyIdentifier, String>,
	map_unique_key_stack: HashMap<UniqueKeyIdentifier, Vec<Frame>>,
}

pub struct CallEvent{
	caller: String,
	callee: String,
}
#[derive(Clone)]
pub struct Frame{
    // 3 values Name, accumulated time, current_time
    name : String,
    accumulated_time : u64,
    time_stamp : u64,
}

impl CallStack{
    pub fn new() -> Self{
        Self{
        map_trace_id: HashMap::new(),
        map_unique_key_stack:  HashMap::new(),
    }
}

    pub fn Enter(&mut self, identifier : UniqueKeyIdentifier ,children_function :String, enter_time : u64) -> Option<(CallEvent, String)> {

        let current_stack = self.map_unique_key_stack.entry(identifier).or_insert_with(|| Vec::new());
		let newFrame =  Frame {
			name: children_function.clone(),
			accumulated_time: 0,
			time_stamp: enter_time
		};
        current_stack.push(newFrame);
		if current_stack.len() > 1{
			let father_function = current_stack[current_stack.len() - 2].name.clone();
			let current_uuid = self.map_trace_id[&identifier].clone();
			return Some((CallEvent{caller: father_function, callee:children_function}, current_uuid));
		}
		let uuid = Uuid::new_v4();
		self.map_trace_id.insert(identifier, uuid.to_string());
		None
    }

    pub fn Exit(&mut self, identifier : UniqueKeyIdentifier, exit_time : u64) -> u64{
        // exit time when a function finishes
		//  current stack
		let current_stack = self.map_unique_key_stack.get_mut(&identifier).expect("not matching identifier in call stack");
		let start_time = current_stack[current_stack.len() -1].time_stamp;
		let total_time = exit_time - start_time;
		if current_stack.len() >= 2{
			let size = current_stack.len();
			current_stack[size - 2].accumulated_time += total_time;
		}
		let accumulated_time_children = current_stack[current_stack.len() -1].accumulated_time;
		let self_time = total_time - accumulated_time_children;
		// eliminate the last value in the stack
		current_stack.pop();
		// remove key in the hashmap if the stack is empty
		if current_stack.len() == 0{
			self.map_unique_key_stack.remove(&identifier);
			self.map_trace_id.remove(&identifier);
		}
		
		self_time
    }

    
}