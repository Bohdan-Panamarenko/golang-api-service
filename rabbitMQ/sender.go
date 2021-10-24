package rabbitMQ

import (
	"os"
	"os/signal"
	"sync"

	"github.com/streadway/amqp"
)

type Queue struct {
	arr []int
}

func NewQueue() *Queue {
	q := &Queue{
		arr: make([]int, 0),
	}

	return q
}

func (q *Queue) Push(item int) {
	q.arr = append(q.arr, item)
}

func (q *Queue) Pop() int {
	item := q.arr[0]
	q.arr = q.arr[1:]
	return item
}

type WorkerPool struct {
	sync.RWMutex
	idQueue *Queue
	size    int
	// workerList map[int]chan float64
}

func NewWorkerPool(poolSize int) *WorkerPool {
	wp := &WorkerPool{
		idQueue: NewQueue(),
		size:    poolSize,
	}

	for i := 1; i <= poolSize; i++ {
		wp.idQueue.Push(i)
	}

	return wp
}

func (wp *WorkerPool) getId() int {
	wp.Lock()
	defer wp.Unlock()
	return wp.idQueue.Pop()
}

func (wp *WorkerPool) putId(id int) {
	wp.Lock()
	defer wp.Unlock()
	wp.idQueue.Push(id)
}

func (wp *WorkerPool) canStart() bool {
	wp.RLock()
	defer wp.RUnlock()
	return len(wp.idQueue.arr) != 0
}

func (wp *WorkerPool) newWorker(jobs <-chan BrockerMessage, wg *sync.WaitGroup, ch *amqp.Channel, queueName string) {
	id := wp.getId()
	defer func() {
		wp.putId(id)
		wg.Done()
	}()

	// fmt.Printf("worker:%d spawning\n", id)

	for {
		select {
		case j := <-jobs:
			// fmt.Printf("worker:%d sleep:%.1f\n", id, j)
			// time.Sleep(time.Duration(int(j * float64(int(time.Second)))))
			err := ch.Publish(
				"",        // exchange
				queueName, // routing key
				false,     // mandatory
				false,     // immediate
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        []byte(j.Msg),
				},
			)
			if err != nil {
				PrintError(err, "Failed to publish a message int sender")
			}
		default:
			return
		}
	}
}

// func (wp *WorkerPool) stats() int {
// 	return wp.size - len(wp.idQueue.arr)
// }

// func statsHandler(wp *WorkerPool) func(w http.ResponseWriter, r *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte(fmt.Sprint(wp.Stats())))
// 	}
// }

// func (wp *WorkerPool) Serve() *http.Server {
// 	r := mux.NewRouter()
// 	r.HandleFunc("/stats", statsHandler(wp)).Methods(http.MethodGet)
// 	srv := http.Server{
// 		Addr:    ":8080",
// 		Handler: r,
// 	}

// 	go func() {
// 		err := srv.ListenAndServe()
// 		if err != nil && err != http.ErrServerClosed {
// 			log.Println("Server exited with error:", err)
// 		}
// 	}()

// 	return &srv
// }

func RunSender(poolSize int, jobs <-chan BrockerMessage) {
	wp := NewWorkerPool(poolSize)
	var wg sync.WaitGroup

	jobsWorkers := make(chan BrockerMessage, 2*poolSize)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		PrintError(err, "Filed to connect to RabbitMQ int sender:")
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		PrintError(err, "Failed to open a channel int sender:")
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		PrintError(err, "Failed to declare a queue int sender:")
		return
	}

	for {
		select {
		case req := <-jobs:
			jobsWorkers <- req
			if wp.canStart() {
				wg.Add(1)
				go wp.newWorker(jobsWorkers, &wg, ch, q.Name)
			}
		case <-interrupt:
			wg.Wait()
			return
		}
	}

}

// func readInput(results chan<- float64, errChan chan<- int) {
// 	reader := bufio.NewReader(os.Stdin)

// 	for {
// 		text, err := reader.ReadString('\n') // read user input
// 		if err != nil {
// 			if err == io.EOF {
// 				errChan <- 1
// 				return
// 			}
// 			log.Println("User input reading err:", err.Error())
// 			continue
// 		}

// 		text = strings.Replace(text, "\n", "", -1) // remove end of line symbol from user input

// 		float, err := strconv.ParseFloat(text, 32)
// 		if err != nil {
// 			log.Println("Float parsing err:", err.Error())
// 		}

// 		results <- float
// 	}
// }
