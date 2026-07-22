package main


import (

	"fmt"
	"os"
)


func Getfunctions(binary_path string) ([]string, error){

	file, err := elf.Open(binary_path)
	
	if err != nil{
		return nil, err
	}

	defer file.close()

	symbols. err :=  file.Symbols()
	 
	if err != nil{
		return nil, err
	}

	var functions []string

	for  _ ,sym := range symbols{
		if elf.ST_TYPE(sym.Info) == elf.STT_FUNC{
			funcs = append(funcs, sym.Name)
		}
		
	}
	return functions,  nil

}
