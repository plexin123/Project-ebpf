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
        cs.Enter(k, "A".into(), 0);
        cs.Enter(k, "B".into(), 10);
        assert_eq!(cs.Exit(k,30),20);
        cs.Enter(k, "C".into(), 15);
        assert_eq!(cs.Exit(k,35),20);
        assert_eq!(cs.Exit(k,50),10);
        
    }

}
