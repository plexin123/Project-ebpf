
use std::collections::HashMap;
use uuid::Uuid;

#[derive(Debug,Clone,Copy,Hash,PartialEq,Eq)]
pub struct UniqueKeyIdentifier{
    pub pid : u32,
    pub generic_key : u32,
}

pub struct CallStack{
	pub map_trace_id   :  HashMap<UniqueKeyIdentifier, String>,
	pub map_unique_key_stack: HashMap<UniqueKeyIdentifier, Vec<Frame>>,
}

// caller is empty when callee is the root of the stack
pub struct CallEvent{
	pub trace_id: String,
	pub caller: String,
	pub callee: String,
}

pub struct ExitEvent{
	pub trace_id: String,
	pub function: String,
	pub self_time: u64,
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

    pub fn enter(&mut self, identifier : UniqueKeyIdentifier ,children_function :String, enter_time : u64) -> CallEvent {

        let current_stack = self.map_unique_key_stack.entry(identifier).or_insert_with(|| Vec::new());
		let new_frame =  Frame {
			name: children_function.clone(),
			accumulated_time: 0,
			time_stamp: enter_time
		};
        current_stack.push(new_frame);
		if current_stack.len() > 1{
			let father_function = current_stack[current_stack.len() - 2].name.clone();
			let current_uuid = self.map_trace_id[&identifier].clone();
			return CallEvent { trace_id: current_uuid, caller: father_function, callee: children_function };
		}
		// root of a new stack -> new trace
		let uuid = Uuid::new_v4().to_string();
		self.map_trace_id.insert(identifier, uuid.clone());
		CallEvent { trace_id: uuid, caller: String::new(), callee: children_function }
    }

    // None when there is no matching enter (e.g. the function was already running when the probes were attached)
    pub fn exit(&mut self, identifier : UniqueKeyIdentifier, exit_time : u64) -> Option<ExitEvent>{
        // exit time when a function finishes
		//  current stack
		let current_stack = self.map_unique_key_stack.get_mut(&identifier)?;
		let frame = current_stack.pop()?;
		let total_time = exit_time - frame.time_stamp;
		// the parent is now the top of the stack
		if let Some(parent) = current_stack.last_mut(){
			parent.accumulated_time += total_time;
		}
		let self_time = total_time - frame.accumulated_time;
		let current_uuid = self.map_trace_id[&identifier].clone();
		// remove key in the hashmap if the stack is empty
		if current_stack.is_empty(){
			self.map_unique_key_stack.remove(&identifier);
			self.map_trace_id.remove(&identifier);
		}

		Some(ExitEvent { trace_id: current_uuid, function: frame.name, self_time })
    }

    
}