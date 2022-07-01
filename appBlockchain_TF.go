package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"runtime"
	"time"
	"unicode"
)

type MessageType int32

const (
	NEWHOST  MessageType = 0
	ADDHOST  MessageType = 1
	ADDLOCK  MessageType = 2
	NEWBLOCK MessageType = 3
	SETBLOCK MessageType = 4
	UPDATEBLOCK MessageType = 5
	PROCOTOL             = "tcp"
	NEWCB                = 1
	LISTCB               = 2
	LISTHOST             = 3
	DEPCB                = 4
	ENVCB                = 5
	LISTACC				 = 6
	LISTDEP				 = 7
	LISTENV				 = 8
)

// utils

// CLEAR SCREEN
var clear map[string]func() //create a map for storing clear funcs

func init() {
    clear = make(map[string]func()) //Initialize it
    clear["linux"] = func() { 
        cmd := exec.Command("clear") //Linux example, its tested
        cmd.Stdout = os.Stdout
        cmd.Run()
    }
    clear["windows"] = func() {
        cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested 
        cmd.Stdout = os.Stdout
        cmd.Run()
    }
}

func CallClear() {
    value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
    if ok { //if we defined a clear func for that platform:
        value()  //we execute it
    } else { //unsupported platform
        panic("Your platform is unsupported! I can't clear terminal screen :(")
    }
}

/****************************BCIP*************************/
var HOSTS []string
var LOCAHOST string

type RequestBody struct {
	Message     string
	MessageType MessageType
}

func GetMessage(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	data, _ := reader.ReadString('\n')
	return strings.TrimSpace(data)
}

func SendMessage(toHost string, message string) {
	conn, _ := net.Dial(PROCOTOL, toHost)
	defer conn.Close()
	fmt.Fprint(conn, message)
}

func RemoveHost(index int, hosts []string) []string {
	n := len(hosts)
	hosts[index] = hosts[n-1]
	hosts[n-1] = ""
	return hosts[:n-1]
}

func RemoveHostByValue(ip string, hosts []string) []string {
	for index, host := range hosts {
		if host == ip {
			return RemoveHost(index, hosts)
		}

	}
	return hosts
}

func Broadcast(newHost string) {
	for _, host := range HOSTS {
		data := append(HOSTS, newHost, LOCAHOST)
		data = RemoveHostByValue(host, data)
		requestBroadcast := RequestBody{
			Message:     strings.Join(data, ","),
			MessageType: ADDHOST,
		}
		broadcastMessage, _ := json.Marshal(requestBroadcast)
		SendMessage(host, string(broadcastMessage))
	}
}

func BroadcastBlock(newBlock Block) {
	for _, host := range HOSTS {
		data, _ := json.Marshal(newBlock)
		requestBroadcast := RequestBody{
			Message:     string(data),
			MessageType: ADDLOCK,
		}
		broadcastMessage, _ := json.Marshal(requestBroadcast)
		SendMessage(host, string(broadcastMessage))
	}
}

func BroadcastBlockUpdate(newBlock Block) {
	for _, host := range HOSTS {
		fmt.Println(host)
		data, _ := json.Marshal(newBlock)
		requestBroadcast := RequestBody{
			Message:     string(data),
			MessageType: UPDATEBLOCK,
		}
		broadcastMessage, _ := json.Marshal(requestBroadcast)
		SendMessage(host, string(broadcastMessage))
	}
}

func BCIPServer(end chan<- int, updateBlocks chan int) {
	ln, _ := net.Listen(PROCOTOL, LOCAHOST)
	defer ln.Close()
	for {
		conn, _ := ln.Accept()
		defer conn.Close()
		request := RequestBody{}
		data := GetMessage(conn)
		_ = json.Unmarshal([]byte(data), &request)
		if request.MessageType == NEWHOST {
			message := strings.Join(append(HOSTS, LOCAHOST), ",")
			requestClient := RequestBody{
				Message:     message,
				MessageType: ADDHOST,
			}
			clientMessage, _ := json.Marshal(requestClient)
			SendMessage(request.Message, string(clientMessage))
			Broadcast(request.Message)
			HOSTS = append(HOSTS, request.Message)
		} else if request.MessageType == ADDHOST {
			HOSTS = strings.Split(request.Message, ",")
		} else if request.MessageType == NEWBLOCK {
			blocksMessage, _ := json.Marshal(localBlockchain.Chain)
			setBlocksRequest := RequestBody{
				Message:     string(blocksMessage),
				MessageType: SETBLOCK,
			}
			setBlocksMessage, _ := json.Marshal(setBlocksRequest)
			SendMessage(request.Message, string(setBlocksMessage))
		} else if request.MessageType == SETBLOCK {
			_ = json.Unmarshal([]byte(request.Message), &localBlockchain.Chain)
			updateBlocks <- 0
		} else if request.MessageType == ADDLOCK {
			block := Block{}
			src := []byte(request.Message)
			json.Unmarshal(src, &block)
			localBlockchain.Chain = append(localBlockchain.Chain, block)
		} else if request.MessageType == UPDATEBLOCK {
			block := Block{}
			src := []byte(request.Message)
			json.Unmarshal(src, &block)
			for i := 0; i < len(localBlockchain.Chain); i++ { 
				if localBlockchain.Chain[i].Hash == block.Hash {
					localBlockchain.Chain[i].Data = block.Data
				}
			}
		}

	}
	end <- 0
}

/************************************BLOCK CHAIN *****************************/

type CuentaBancaria struct {
	Nombre     string
	DNI        string
	Clave      string
	Saldo      float64 //Inicializar en 0
	DNIDestino string  //Para envíos ""
}

type Block struct {
	Index        int
	Timestamp    time.Time
	Data         CuentaBancaria
	PreviousHash string
	Hash         string
}

type Blockchain struct {
	Chain []Block
}

func (Blockchain *Blockchain) GetLatestBlock() Block {
	n := len(Blockchain.Chain)
	return Blockchain.Chain[n-1]
}

func (Blockchain *Blockchain) GetBlock(dni string) Block { //deber retornar bloque o error

	n := len(Blockchain.Chain)

	for i := 0; i < n; i++ { //Verificar indices (Bloque genesis)
		if len(Blockchain.Chain[i].Data.Nombre) > 0 {

			if dni == Blockchain.Chain[i].Data.DNI {
				return Blockchain.Chain[i]
			}
		}
	}
	block := Block{}
	return block // Si no se encuentra el bloque de la cuenta bancaria -> retornar bloque vacío
}

func (Blockchain *Blockchain) Deposito(dni string, monto float64) Block{
	blockOrigen := Blockchain.GetBlock(dni)
	blockToBroadcast := Block{}
	if len(blockOrigen.Data.DNI) > 0 {// Si existe dni del usuario para depositar

		for i := 0; i < len(Blockchain.Chain); i++ {
			if Blockchain.Chain[i].Hash == blockOrigen.Hash {

				Blockchain.Chain[i].Data.Saldo += monto
				fmt.Println("Se depositó el dinero con éxito")
				blockToBroadcast = Blockchain.Chain[i]
				//BroadcastBlock(Blockchain.Chain[i])
			}
		}
	}
	return blockToBroadcast
}

func (Blockchain *Blockchain) EnviarDinero(dniEmisor string, claveEmisor string, monto float64, dniReceptor string) int32{
	blockEmisor := Blockchain.GetBlock(dniEmisor)
	blockReceptor := Blockchain.GetBlock(dniReceptor)

	if blockEmisor.Data.Clave != claveEmisor {
		fmt.Println("clave inválida o usuario emisor inexistente")
		return -1
	}

	if blockEmisor.Data.Saldo < monto{
		fmt.Println("Saldo insuficiente")
		return -1
	}

	if len(blockEmisor.Data.DNI) > 0 && len(blockReceptor.Data.DNI) > 0 {

		for i := 0; i < len(Blockchain.Chain); i++ {

			if Blockchain.Chain[i].Data == blockEmisor.Data {

				Blockchain.Chain[i].Data.Saldo -= monto
				BroadcastBlockUpdate(Blockchain.Chain[i])
				fmt.Println(" Se extrajo el dinero de la cuenta del emisor")

			} else if Blockchain.Chain[i].Data == blockReceptor.Data {

				Blockchain.Chain[i].Data.Saldo += monto
				BroadcastBlockUpdate(Blockchain.Chain[i])
				fmt.Println(" Se depositó el dinero a la cuenta del receptor")
			}
		}
		fmt.Println("Transferencia exitosa")
		return 1
	} else {
		fmt.Println("Error al ingresar alguno de los DNIs")
		return -1
	}
}

func (Blockchain *Blockchain) addBlock(block Block) Block {
	block.Timestamp = time.Now()
	block.Index = Blockchain.GetLatestBlock().Index + 1
	block.PreviousHash = Blockchain.GetLatestBlock().Hash
	block.Hash = block.CalculateHash()
	Blockchain.Chain = append(Blockchain.Chain, block)
	return block
}

func (block *Block) CalculateHash() string {
	src := fmt.Sprintf("%d-%s-%s", block.Index, block.Timestamp.String(), block.Data)
	return base64.StdEncoding.EncodeToString([]byte(src))
}

func (Blockchain *Blockchain) CreateGenesisBlock() Block {
	block := Block{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         CuentaBancaria{},
		PreviousHash: "0",
	}
	block.Hash = block.CalculateHash()
	return block
}

func createBlockchain() Blockchain {
	bc := Blockchain{}
	genesisBlock := bc.CreateGenesisBlock()
	bc.Chain = append(bc.Chain, genesisBlock)
	return bc
}

func printBlockChain() {
	blocks := localBlockchain.Chain[1:]

	fmt.Printf("\n")
	fmt.Printf("-------- Blockchain's Block List --------\n")

	for index, block := range blocks {
		cuentaBancaria := block.Data

		fmt.Printf("------Bloque No. %d----\n", index+1)
		if len(cuentaBancaria.Nombre) > 0 {
			fmt.Printf("--- REGISTRO DE CUENTA BANCARIA ---\n")
		}
		if len(cuentaBancaria.DNIDestino) > 0 {
			fmt.Printf("--- REGISTRO DE ENVÍO ---\n")
		}
		if len(cuentaBancaria.Nombre) == 0 && len(cuentaBancaria.DNIDestino) == 0 {
			fmt.Printf("--- REGISTRO DE DEPÓSITO ---\n")
		}

		fmt.Printf("\n")

		if len(cuentaBancaria.Nombre) > 0 {
			fmt.Printf("\tNombre: %s", cuentaBancaria.Nombre)
		}
		fmt.Printf("\tDNI: %s", cuentaBancaria.DNI) //DNI SIEMPRE SE IMPRIME
		if len(cuentaBancaria.Clave) > 0 {
			fmt.Printf("\tClave: %s", cuentaBancaria.Clave)
		}
		if len(cuentaBancaria.DNIDestino) > 0 {
			fmt.Printf("\tDNI Destino: %s", cuentaBancaria.DNIDestino)
		}
		fmt.Printf("\tSaldo: %.2f \n", cuentaBancaria.Saldo)

		fmt.Println("\n")

		fmt.Println("\tTimeStamp: " + block.Timestamp.String())
		fmt.Println("\tHash code: " + block.Hash)
		fmt.Println("\tPrevious Hash code: " + block.PreviousHash)
		fmt.Println("\n")
		
	}
}

func printAccounts() {
	blocks := localBlockchain.Chain[1:]

	fmt.Printf("\n")
	fmt.Printf("-------- Blockchain's Accounts Block List --------\n")
	for index, block := range blocks {
		cuentaBancaria := block.Data

		if len(cuentaBancaria.Nombre) > 0 {
			fmt.Printf("------Bloque No. %d----\n", index+1)
			fmt.Printf("\tNombre: %s", cuentaBancaria.Nombre)
			fmt.Printf("\tDNI: %s", cuentaBancaria.DNI)
			fmt.Printf("\tSaldo: %.2f \n", cuentaBancaria.Saldo)
			fmt.Println("\n")
			fmt.Println("\tFecha de apertura: " + block.Timestamp.String())
			fmt.Println("\tHash code: " + block.Hash)
			fmt.Println("\tPrevious Hash code: " + block.PreviousHash)
			fmt.Println("\n")
		}
	}
}

func printDepositos() {
	blocks := localBlockchain.Chain[1:]

	fmt.Printf("\n")
	fmt.Printf("-------- Blockchain's Deposits Block List --------\n")
	for index, block := range blocks {
		cuentaBancaria := block.Data

		if len(cuentaBancaria.Nombre) == 0 && len(cuentaBancaria.DNIDestino) == 0 {
			fmt.Printf("------Bloque No. %d----\n", index+1)
			fmt.Printf("\tDNI: %s", cuentaBancaria.DNI)
			fmt.Printf("\nDinero depositado: %.2f \n", cuentaBancaria.Saldo)
			fmt.Println("\n")
			fmt.Println("\tFecha del depósito: " + block.Timestamp.String())
			fmt.Println("\tHash code: " + block.Hash)
			fmt.Println("\tPrevious Hash code: " + block.PreviousHash)
			fmt.Println("\n")
		}
	}
}

func printTransfers() {
	blocks := localBlockchain.Chain[1:]

	fmt.Printf("\n")
	fmt.Printf("-------- Blockchain's Transfers Block List --------\n")
	for index, block := range blocks {
		cuentaBancaria := block.Data

		if len(cuentaBancaria.DNIDestino) > 0 {
			fmt.Printf("------Bloque No. %d----\n", index+1)
			fmt.Printf("\tDNI: %s", cuentaBancaria.DNI)
			fmt.Printf("\tDNI Destino: %s", cuentaBancaria.DNIDestino)
			fmt.Printf("\nDinero transferido: %.2f \n", cuentaBancaria.Saldo)
			fmt.Println("\n")
			fmt.Println("\tFecha del depósito: " + block.Timestamp.String())
			fmt.Println("\tHash code: " + block.Hash)
			fmt.Println("\tPrevious Hash code: " + block.PreviousHash)
			fmt.Println("\n")
		}
	}
}

func printHosts() {
	fmt.Println("--------------HOST -----")
	const first = 0
	fmt.Printf("\t%s (Your host)\n", LOCAHOST)
	for _, host := range HOSTS {
		fmt.Printf("\t%s\n", host)
	}
}

var localBlockchain Blockchain

func (Blockchain *Blockchain) cbExiste(dni string) bool {
	for i := 0; i < len(Blockchain.Chain); i++ {
		if Blockchain.Chain[i].Data.DNI == dni {
			return true
		}
	}
	return false
}

func isInt(s string) bool {
    for i, c := range s {
    	if i != 8 && i != 9 {
        	if !unicode.IsDigit(c) {
        	    return false
        	}
    	}
    }
    return true
}

func main() {
	var dest string
	end := make(chan int)
	updatedBlocks := make(chan int)

	CallClear()
	fmt.Print("Ingrese su host: ")
	fmt.Scanf("%s\n", &LOCAHOST)
	fmt.Print(
		"Ingrese el host de destino (No ingresar si es el primer registro): ")
	fmt.Scanf("%s\n", &dest)
	go BCIPServer(end, updatedBlocks)
	localBlockchain = createBlockchain()
	if dest != "" {
		requestBody := &RequestBody{
			Message:     LOCAHOST,
			MessageType: NEWHOST,
		}

		requestMessage, _ := json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		requestBody.MessageType = NEWBLOCK
		requestMessage, _ = json.Marshal(requestBody)
		SendMessage(dest, string(requestMessage))
		<-updatedBlocks
	}

	
	var action int
	fmt.Println("Bienvenido!")
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("1. Registro de Usuario\n2. Lista de Registros\n3. Lista de Hosts\n4. Depositar Dinero\n5. Enviar Dinero\n 6. Lista de Cuentas Bancarias\n 7. Lista de Depósitos\n 8. Lista de Transferencias\n")
		fmt.Print("Ingrese opción 1|2|3|4|5|6|7|8 \n")
		fmt.Scanf("%d\n", &action)

		// 1 Nueva cuenta bancaria
		if action == NEWCB {
			CallClear() // cls
			cuentaBancaria := CuentaBancaria{}

			fmt.Println("---Registro---")
			fmt.Print("Ingrese su Nombre: ")
			cuentaBancaria.Nombre, _ = in.ReadString('\n')

			dniExiste:=false
			validDNILength:=false
			onlyNumbersDNI:=false

			for !validDNILength || dniExiste || !onlyNumbersDNI {
				fmt.Print("Ingrese su DNI: ")
				cuentaBancaria.DNI, _ = in.ReadString('\n')
				// se valida 1 vez por ingreso de clave
				//// 10 dígitos
				if len(cuentaBancaria.DNI) != 10 { // length + 2
					validDNILength = false
					fmt.Print("Error: el DNI debe tener 8 caracteres. ")
					continue
				} else {
					validDNILength = true
				}
				//// solo números
				onlyNumbersDNI = isInt(cuentaBancaria.DNI)
				if !onlyNumbersDNI {
					fmt.Print("Error: el DNI solo debe tener números. ")
					continue
				}
				//se recorre la lista para bucar match
				for i := 0; i < len(localBlockchain.Chain); i++ {
					if localBlockchain.Chain[i].Data.DNI == cuentaBancaria.DNI {
						dniExiste = true
						fmt.Print("Error: ese DNI ya se encuentra registrado. ")
						break
					} else {
						dniExiste = false
					}
				}
			}

			fmt.Print("Ingrese su Clave: ")
			cuentaBancaria.Clave, _ = in.ReadString('\n')
			cuentaBancaria.Saldo = 0.00
			cuentaBancaria.DNIDestino = ""
			newBlock := Block{
				Data: cuentaBancaria,
			}
			blockToBroadcast := localBlockchain.addBlock(newBlock)

			BroadcastBlock(blockToBroadcast)

			fmt.Println("Se ha registrado satisfactoriamente!")
		    time.Sleep(3 * time.Second)
		    CallClear()

		} else if action == LISTCB { // 2 listar blockchain
			CallClear() // cls
			printBlockChain()

		} else if action == LISTHOST { // 3 listar hosts
			CallClear() // cls
			printHosts()

		} else if action == DEPCB { // 4 Depositar o extraer dinero
			CallClear() // cls

			cuentaBancaria := CuentaBancaria{}

			fmt.Println(("---Depositar dinero---"))

			cuentaBancaria.Nombre = ""
			cuentaBancaria.Clave = ""
			cuentaBancaria.DNIDestino = ""

			fmt.Print("Ingrese su DNI: ")
			dni, _ := in.ReadString('\n')
			cuentaBancaria.DNI = dni

			fmt.Print("Ingrese la cantidad a depositar:")
			tmp, _ := in.ReadString('\n')
			input := strings.TrimSpace(tmp)
			saldo, _ := strconv.ParseFloat(input, 64)
			cuentaBancaria.Saldo = saldo

			newBlock := Block{
				Data: cuentaBancaria,
			}
			
			// obtener bloque cuenta bancaria a depositar
			bloqueCuentaBancaria := localBlockchain.GetBlock(dni)

			if len(bloqueCuentaBancaria.Data.DNI) > 0 {

				// Depositar dinero a bloque cuenta bancaria
				blockToBroadcastUpdate := localBlockchain.Deposito(dni, saldo)
				BroadcastBlockUpdate(blockToBroadcastUpdate)

				// Registrar bloque del registro del depósito
				blockToBroadcast := localBlockchain.addBlock(newBlock)
				BroadcastBlock(blockToBroadcast)
				fmt.Println("Se ha creado el registro del depósito satisfactoriamente!")
			} else {
				fmt.Println("Error: El DNI ingresado no existe.")
			}

			time.Sleep(3 * time.Second)
			CallClear() // cls

		} else if action == ENVCB { // 5 enviar dinero de una cuenta a otra

			cuentaBancaria := CuentaBancaria{}

			fmt.Println(("---Envío---"))

			cuentaBancaria.Nombre = ""

			fmt.Print("Ingrese su DNI: ")
			dni, _ := in.ReadString('\n')
			cuentaBancaria.DNI = dni

			fmt.Print("Ingrese su Clave: ")
			cuentaBancaria.Clave, _ = in.ReadString('\n')

			fmt.Print("Ingrese el DNI de Destino: ")
			dniDest, _ := in.ReadString('\n')
			cuentaBancaria.DNIDestino = dniDest

			fmt.Print("Ingrese el Saldo a Depositar:")
			tmp, _ := in.ReadString('\n')
			input := strings.TrimSpace(tmp)
			saldo, _ := strconv.ParseFloat(input, 64)
			cuentaBancaria.Saldo = saldo

			newBlock := Block{
				Data: cuentaBancaria,
			}

			//Llamar función Envío(dni,saldo, dniDest) CON ESTO SE ACTUALIZA EL BLOQUE DE USUARIO QUE ENVÍA Y QUE RECIBE
			isValid := localBlockchain.EnviarDinero(dni, cuentaBancaria.Clave, saldo, dniDest)

			if isValid == 1 {
				// Registrar bloque del registro del depósito
				blockToBroadcast := localBlockchain.addBlock(newBlock)
				BroadcastBlock(blockToBroadcast)
				fmt.Println("Se ha enviado el dinero satisfactoriamente!")
				time.Sleep(3 * time.Second)
				CallClear() // cls
			} else {
				fmt.Println("Error: no se efectuó la transacción de envío")
				time.Sleep(3 * time.Second)
				CallClear() // cls 
			}
			
		} else if action == LISTACC { // Listar cuentas bancarias
			CallClear() // cls
			printAccounts()
		} else if action == LISTDEP {
			CallClear() // cls
			printDepositos()
		} else if action == LISTENV {
			CallClear() // cls
			printTransfers()
		}

		//else if action == 6 {
		//	fmt.Println("len(Blockchain.Chain)")
		//	len(Blockchain.Chain)
		//	len(Blockchain.Chain[i].Data.Nombre)
		//	Blockchain.Chain[i].Data.DNI
		//}

	}
	<-end
 
}
