pub mod callstack;
pub fn add(left: u64, right: u64) -> u64 {
    left + right
}
pub fn subtract(left: i64, right: i64) -> i64{
    right - left
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn it_works() {
        let result = add(2, 2);
        assert_eq!(result, 4);
    }
    #[test]
    fn it_works2(){
        let result2 =subtract(3,2);
        assert_eq!(result2, -1);
    }
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
        assert_eq!((a.function.as_str(), a.self_time), ("A", 10));

    }

    #[test]
    fn exit_without_enter_is_none(){
        let mut cs =callstack::CallStack::new();
        let k = callstack::UniqueKeyIdentifier{pid : 1, generic_key: 2};
        assert!(cs.exit(k, 10).is_none());
    }

}
