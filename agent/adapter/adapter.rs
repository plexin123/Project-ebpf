
trait Adapter{
    fn new() -> (FunctionStats,Self)
    fn scan(&mut self) -> (sym, ex);
    fn map(&mut self, sym: Sym, ex : Ex) -> (reader)
    fn reader(&mut self, reader : Reader) -> (FunctionEvent)
}

struct RustAdapter {
    register_map : HashMap<uint64,String>
    map_of_functions: HashMap<String, FunctionStats>
    links : Vec<Links> 
}


impl Adapter for RustAdapter{
    fn new() -> (Self){
        Self{
            register_map : HashMap::new(),
            map_of_functions: HashMap::new(),
            links : Vec::new(),
    };
}
    fn scan(&mut self) -> (sym, ex){
        // leer los simbolos del binario que se va a perfilar
        // direccion -> name of the function 
    }

    fn map(&mut self, sym :Sym, ex: Ex) -> (reader){

    }

    fn reader(&mut self, reader: Reader) -> {
        
    }
}