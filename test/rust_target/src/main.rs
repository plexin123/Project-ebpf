use std::thread;
use std::time::Duration;

#[inline(never)]
fn func1(){

    let mut count = 0;

    for i in (1..10){
        count +=  i
    }
    
    func2();
}


#[inline(never)]
fn func2(){
    thread::sleep(Duration::from_secs(2));
    let bytes : &[u8;4] = &[3,4,6,7];
    let mut counter = 0;
    for byte in bytes{
        counter += byte
    }
    print!("{}",counter);
    func3()

}
#[inline(never)]
fn func3(){
    thread::sleep(Duration::from_secs(4));
    let word = "Alana";  
    let mut vector = Vec::new();
    for mut character in word.bytes(){
        character = character + 1;
        vector.push(character as char);
    }
    print!("{:?}", vector);
    
}
#[inline(never)]
fn func4(){
    let mut _count = 0;
    for i in (1..10){
        _count +=  i;
    }
    print!("{}",_count);
}


fn main() {
   let handle = std::thread::spawn(func4);
   // await that thread finishes
   func1();
   handle.join().unwrap(); 
}
