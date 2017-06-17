package ep

import (
    "io"
    "net"
    "context"
    "encoding/gob"
)

var _ = registerGob(exchange{})

const (
    sendGather = 1
    sendScatter = 2
    sendBroadcast = 3
    sendPartition = 4
)

// Scatter returns an exchange Runner that scatters its input uniformly to
// all other nodes such that the received datasets are dispatched in a round-
// robin to the nodes.
func Scatter() Runner {
    return nil
}

// Gather returns an exchange Runner that gathers all of its input into a
// single node. In all other nodes it will produce no output, but on the main
// node it will be passthrough from all of the other nodes
func Gather() Runner {
    return nil
}

// Broadcast returns an exchange Runner that duplicates its input to all
// other nodes. The output will be effectively a union of all of the inputs from
// all nodes (order not guaranteed)
func Broadcast() Runner {
    return nil
}

// exchange is a Runner that exchanges data between peer nodes
type exchange struct {
    Uid string
    SendTo int

    encs []*gob.Encoder // encoders to all destination connections
    decs []*gob.Decoder // decoders from all source connections
    conns []net.Conn // all open connections (used for closing)
    encsNext int // Encoders Round Robin next index
    decsNext int // Decoders Round Robin next index
}

func (ex *exchange) Run(ctx context.Context, inp, out chan Dataset) (err error) {
    defer func() { ex.Close(err) }()
    err = ex.Init(ctx)
    if err != nil {
        return
    }

    // send the local data to the distributed target nodes
    go func() {
        for data := range inp {
            err = ex.Send(data)
            if err != nil {
                return
            }
        }
    }()

    // listen to all nodes for incoming data
    var data Dataset
    for {
        data, err = ex.Receive()
        if err != nil {
            return
        }

        out <- data
    }
}

// Send a dataset to destination nodes
func (ex *exchange) Send(data Dataset) error {
    switch ex.SendTo {
    case sendScatter:
        return ex.EncodeNext(data)
    case sendPartition:
        return ex.EncodePartition(data)
    default:
        return ex.EncodeAll(data)
    }
}

func (ex *exchange) Receive() (Dataset, error) {
    data := NewDataset()
    err := ex.DecodeNext(data)
    return data, err
}

// Close all open connections. If an error object is supplied, it's first
// encoded to all connections before closing.
func (ex *exchange) Close(err error) error {
    var errOut error
    if err != nil {
        panic("not implemented")
        // errOut = ex.EncodeAll(err)
    }

    for _, conn := range ex.conns {
        err1 := conn.Close()
        if err1 != nil {
            errOut = err1
        }
    }

    return errOut
}

// Encode an object to all destination connections
func (ex *exchange) EncodeAll(e Dataset) (err error) {
    for _, enc := range ex.encs {
        err1 := enc.Encode(e)
        if err1 != nil {
            err = err1
        }
    }

    return err
}

// Encode an object to the next destination connection in a round robin
func (ex *exchange) EncodeNext(e Dataset) error {
    if len(ex.encs) == 0 {
        return io.ErrClosedPipe
    }

    ex.encsNext = (ex.encsNext + 1) % len(ex.encs)
    return ex.encs[ex.encsNext].Encode(e)
}

// Encode an object to a destination connection selected by partitioning
func (ex *exchange) EncodePartition(e Dataset) error {
    return nil
}

// Decode an object from the next source connection in a round robin
func (ex *exchange) DecodeNext(e Dataset) error {
    if len(ex.decs) == 0 {
        return io.EOF
    }

    i := (ex.decsNext + 1) % len(ex.decs)
    err := ex.decs[i].Decode(e)
    if err == io.EOF {
        // remove the current decoder and try again
        ex.decs = append(ex.decs[:i], ex.decs[i + 1:]...)
        return ex.DecodeNext(e)
    } else if err != nil {
        return err
    }

    ex.decsNext = i
    return nil
}

// initialize the connections, encoders & decoders
func (ex *exchange) Init(ctx context.Context) error {
    var err error
    var isThisTarget bool // is this node also a destination?

    allNodes := ctx.Value("ep.AllNodes").([]string)
    thisNode := ctx.Value("ep.ThisNode").(string)
    masterNode := ctx.Value("ep.MasterNode").(string)
    dist := ctx.Value("ep.Distributer").(interface {
        Connect(addr, uid string) (net.Conn, error)
    })

    targetNodes := allNodes
    if ex.SendTo == sendGather {
        targetNodes = []string{masterNode}
    }

    // open a connection to all target nodes
    var conn net.Conn
    connsMap := map[string]net.Conn{}
    for _, n := range targetNodes {
        isThisTarget = isThisTarget || n == thisNode // TODO: short-circuit
        conn, err = dist.Connect(n, ex.Uid)
        if err != nil {
            return err
        }

        connsMap[n] = conn
        ex.conns = append(ex.conns, conn)
        ex.encs = append(ex.encs, gob.NewEncoder(conn))
    }

    // if we're also a destination, listen to all nodes
    for i := 0; isThisTarget && i < len(allNodes); i += 1 {
        n := allNodes[i]

        // if we already established a connection to this node from the targets,
        // re-use it. We don't need 2 uni-directional connections.
        if connsMap[n] != nil {
            ex.decs = append(ex.decs, gob.NewDecoder(connsMap[n]))
            continue
        }

        conn, err = dist.Connect(n, ex.Uid)
        if err != nil {
            return err
        }

        ex.conns = append(ex.conns, conn)
        ex.decs = append(ex.decs, gob.NewDecoder(conn))
    }
    
    return nil
}