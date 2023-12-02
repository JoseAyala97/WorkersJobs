// declara paquete
package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// struct equivalente a las clases - clase Job
type Job struct {
	Name   string
	Delay  time.Duration
	Number int //para calccular serie fibonacci
}

// clase para worker
type Worker struct {
	Id         int
	JobQueue   chan Job      //Se usan los propios tipos - clases
	WorkerPool chan chan Job //Concepto canal de canales
	QuitChan   chan bool     //En caso de que decidan cerrar los worker
}

// Clase Dispatcher - encargado de enviar toda la información a los workers
type Dispatcher struct {
	WorkerPool chan chan Job //Canal de canales de Job
	MaxWorkers int           //Evitará tener millones de concurrencias ejecutandose al mismo tiempo
	JobQueue   chan Job      //Cola de trabajo que enviará esos datos
}

// Constructor de la clase Worker
func NewWorker(id int, workerPool chan chan Job) *Worker {
	return &Worker{
		//estructura que se le da a ese struct - forma en la cual se usa esa clase
		Id:         id,
		JobQueue:   make(chan Job),
		WorkerPool: workerPool,
		QuitChan:   make(chan bool),
	}
}

// Metodo para el worker - acción que se ejecutará - con función Start()
func (w Worker) Start() {
	//Función anonima debe tener () en el corchete final
	go func() {
		//For para iterar de manera indefinida
		for {
			//Hace lectura del WorkerPool nos viene de JobQueue //Lee de JobQueue y pone disponibe el WorkerPool
			w.WorkerPool <- w.JobQueue
			//Estructura para agregar multiplexación //Similar a u switch
			select {
			//Declaración de casos
			case job := <-w.JobQueue: //estamos leyendo un trabajo de la cola JobQueue
				//Imprime que worker esta siendo ejecutado
				fmt.Printf("Worker with id %d Started\n", w.Id)
				//Se usará para que realice los jobs
				fib := Fibonacci(job.Number)
				//Una vez hecho lo anterior, se pondrá a dormir el programa con el Delay dado
				time.Sleep(job.Delay)
				//Se imprime que el trabajo ha sido terminado
				fmt.Printf("Worker with id %d Finished with result %d", w.Id, fib)
				//Caso donde se llama para quitar el trabajo o que el worker se cierre
			case <-w.QuitChan:
				//Imprimi que este ha sido finalizado
				fmt.Printf("Worker with id %d Stopped\n", w.Id)
			}
		}
	}()
}

// Se declara función Stop()
func (w Worker) Stop() {
	go func() {
		//Se llama al canal con el valor de true, significa que dejará de trabajar
		w.QuitChan <- true
	}()
}

// Se delcara función Fibonacci //Recibe parametro de tipo entero y devuelve un entero
// Se usará para que realice los jobs
func Fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return Fibonacci(n-1) + Fibonacci(n-2)
}

// Constructor de la clase Dispatcher //recibe una cola de tipo trabajo, cantidad maxima de trabajodres y devuelve un Dispatcher
func NewDispatcher(jobQueue chan Job, maxWorkers int) *Dispatcher {
	//Se crea para ayudar a saber cuando estammos enviando Jobs a nuestros trabajadores y se usará un bufferChannel
	worker := make(chan chan Job, maxWorkers)

	return &Dispatcher{
		JobQueue:   jobQueue,
		MaxWorkers: maxWorkers,
		WorkerPool: worker,
	}
}

// Método que realizará encolamiento con función Dispatch()
func (d *Dispatcher) Dispatch() {
	// Iteración con un for
	for {
		// Creación de casos
		select {
		// Lee de JobQueue
		case job := <-d.JobQueue:
			// Función anónima con paréntesis al final para invocarla
			go func() {
				// Nos devuelve el worker pool que se necesita para encolar el trabajo
				workerJobQueue := <-d.WorkerPool
				// Se le envía el Job
				workerJobQueue <- job
			}()
		}
	}
}

// Función Run - creará nuestros trabajadores
func (d *Dispatcher) Run() {
	//Iterar a traves de la cantidad maxima de worker que tenemos
	for i := 0; i < d.MaxWorkers; i++ {
		//Se crear worker nuevo usando nuestro contructor
		worker := NewWorker(i, d.WorkerPool)
		//Se usa función Start para iniciar ese work
		worker.Start()
	}
	//Se usa este dispatch como si fuera una goruntime
	go d.Dispatch()
}

// Encargada de manejar todo lo que el servidor tiene que procesar
// Se usan funciones http
func RequestHandler(w http.ResponseWriter, r *http.Request, jobQueue chan Job) {
	//Se valida que metodo sea igual a POST// en caso de ser GET, PUT o DELETE no lo tomará
	if r.Method != "POST" {
		//en caso de que no, se enviará información de que el metood no será capaz de manejar esto
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	//se crean variables o valores para crear el job
	//Parsea un string y convertirlo en una oración para que sirva como delay }
	//Formvalue permite acceder a todos los valores que se estan enviando en ese request
	delay, err := time.ParseDuration(r.FormValue("delay"))
	//Go no tiene manejo de excepciones por lo tanto se crean las validaciones como diferente de nulo
	if err != nil {
		//Se crea una respuesta de error
		http.Error(w, "Invalide Delay", http.StatusBadRequest)
		//Se hace el return de la función para que no se sigan haciendo los procedimientos posteriores
		return
	}

	//Atoi nos permite pasar un string a un entero
	//Formvalue permite acceder a todos los valores que se estan enviando en ese request
	value, err := strconv.Atoi(r.FormValue("value"))
	if err != nil {
		//Se crea una respuesta de error
		http.Error(w, "Invalide Value", http.StatusBadRequest)
		//Se hace el return de la función para que no se sigan haciendo los procedimientos posteriores
		return
	}

	//Se valida que el string sea diferente a un string vacio
	name := r.FormValue("name")
	if name == "" {
		//Se crea una respuesta de error
		http.Error(w, "Invalide Name", http.StatusBadRequest)
		//Se hace el return de la función para que no se sigan haciendo los procedimientos posteriores
		return
	}
	//se crea el job como tal, dando los valores con las variables anteriormente creadas
	job := Job{Name: name, Delay: delay, Number: value}
	//se usa este canal para realizar encolamiento de las peticiones
	jobQueue <- job
	//se escribe al cliente que toda la petición ha sido procesada de manera correcta
	w.WriteHeader(http.StatusCreated)
}

// Función main donde vive el servidor web
func main() {
	//se definen constantes
	const (
		//Cantidad maxima de trabajadores
		maxWorkers = 4
		//cantidad de trabajos que se podrán hacer simultaneamente
		maxQueueSize = 20
		//Puerto que se declara o donde será escuchado el servicio
		port = ":8081"
	)
	//Buffered Channel - se le da como tamaño el maxQueueSize
	jobQueue := make(chan Job, maxQueueSize)
	//se crea nuevo dispatcher con el Jobqueue, que será el mismo creado anteriormente y se le pasa el maxworker creado anteriormente con constante
	dispatcher := NewDispatcher(jobQueue, maxWorkers)
	//Se crea usa en Run Para que este pueda ejecutarse
	dispatcher.Run()
	//creación
	//http://localhost:8081/fib
	//Se crea o declara la ruta con el /fib
	http.HandleFunc("/fib", func(w http.ResponseWriter, r *http.Request) {
		//Se llama a la función con los parametros anteriormente definidos
		RequestHandler(w, r, jobQueue)
	})
	//En caso de que deje de funcionar este nos mostrará qué esta pasando
	log.Fatal(http.ListenAndServe(port, nil))

}
