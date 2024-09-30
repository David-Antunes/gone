package graphDB

import (
	"context"
	"errors"
	"net"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type dbConn struct {
	uri            string
	user           string
	password       string
	conn           neo4j.DriverWithContext
	ctx            context.Context
	connectedNodes map[string]string
}

var conn *dbConn

func StartConnection(uri string, user string, password string) {
	ctx := context.Background()
	// URI examples: "neo4j://localhost", "neo4j+s://xxx.databases.neo4j.io"
	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.NoAuth())

	if err != nil {
		panic(err)
	}

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		panic(err)
	}

	conn = &dbConn{uri, user, password, driver, ctx, make(map[string]string)}
	err = prepareDatabase()
	if err != nil {
		panic(err)
	}
}

func query(query string, args map[string]any) (*neo4j.EagerResult, error) {
	result, err := neo4j.ExecuteQuery(conn.ctx, conn.conn, query,
		args, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	return result, err
}

func prepareDatabase() error {
	_, err := neo4j.ExecuteQuery(conn.ctx, conn.conn,
		`MATCH (n) DETACH DELETE n`,
		map[string]any{},
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		return err
	}

	_, err = neo4j.ExecuteQuery(conn.ctx, conn.conn,
		`CREATE CONSTRAINT uniq_node_id IF NOT EXISTS
		FOR (n:Node)
		REQUIRE n.id IS UNIQUE
		`,
		map[string]any{}, neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))

	if err != nil {
		return err
	}

	return nil
}

// NodeId is the mac Address of the node
func AddNode(macAddr string, routerId string) error {
	_, ok := conn.connectedNodes[net.HardwareAddr(macAddr).String()]

	if !ok {
		conn.connectedNodes[net.HardwareAddr(macAddr).String()] = routerId

		_, err := query("CREATE (n:Node {id: $id})",
			map[string]any{
				"id": net.HardwareAddr(macAddr).String(),
			})

		if err != nil {
			return err
		}
		return AddPath(net.HardwareAddr(macAddr).String(), routerId, net.HardwareAddr(macAddr).String(), 0)
	}
	return errors.New("node already exists")
}

func RemoveNode(id string) error {
	delete(conn.connectedNodes, net.HardwareAddr(id).String())
	_, err := query(`MATCH (n:Node {id: $id}) DETACH DELETE n`,
		map[string]any{
			"id": net.HardwareAddr(id).String(),
		})
	if err != nil {
		return err
	}
	return nil
}

func AddRouter(id string) error {
	_, err := query("CREATE (n:Node {id: $id})",
		map[string]any{
			"id": id,
		})

	if err != nil {
		return err
	}
	return nil
}

func RemoveRouter(id string) {
	_, err := query("MATCH (n:Node {id: $id}) DETACH DELETE n",
		map[string]any{
			"id": id,
		})

	if err != nil {
		panic(err)
	}
	nodes := make([]string, 0, len(conn.connectedNodes))
	for macAddr, routerId := range conn.connectedNodes {
		if routerId == id {
			nodes = append(nodes, macAddr)
		}
	}
	for _, node := range nodes {
		delete(conn.connectedNodes, node)
		_, err = query("MATCH (n:Node {id: $id}) DETACH DELETE n",
			map[string]any{
				"id": node,
			})
	}
}

//func FindPath(src string, dest string) []string {
//
//	if src == dest {
//		panic("This shouldn't happen.")
//	}
//
//	result, err := query(
//		`MATCH (from:Node { id:$src }), (to:Node { id: $dest}) , path = (from)-[:COSTS*]-(to)
//		RETURN [ n in nodes(path) | n.id ] AS shortestPath,
//		reduce(weight = 0, r in relationships(path) | weight+r.weight) AS totalDistance
//		ORDER BY totalDistance ASC
//		LIMIT 1`,
//		map[string]any{
//			"src":  src,
//			"dest": dest,
//		})
//
//	if err != nil {
//		return make([]string, 0)
//	}
//
//	if len(result.Records) == 0 {
//		return make([]string, 0)
//	}
//	path := result.Records[0].AsMap()["shortestPath"].([]interface{})
//
//	newPath := make([]string, 0, len(path))
//	for _, router := range path {
//		newPath = append(newPath, router.(string))
//	}
//	return newPath
//}

func FindPathToRouter(src string, dest string) ([]string, int) {

	if src == dest {
		return []string{src}, 0
	}
	result, err := query(
		`MATCH (from:Node { id:$src }), (to:Node { id: $dest}) , path = (from)-[:COSTS*]-(to)
		RETURN [ n in nodes(path) | n.id ] AS shortestPath,
		reduce(weight = 0, r in relationships(path) | weight+r.weight) AS totalDistance
		ORDER BY totalDistance ASC
		LIMIT 1`,
		map[string]any{
			"src":  src,
			"dest": net.HardwareAddr(dest).String(),
		})

	if err != nil {
		return make([]string, 0), 0
	}

	if len(result.Records) == 0 {
		return make([]string, 0), 0
	}
	path := result.Records[0].AsMap()["shortestPath"].([]interface{})
	distance := result.Records[0].AsMap()["totalDistance"].(int64)
	newPath := make([]string, 0, len(path))
	for _, router := range path {
		newPath = append(newPath, router.(string))
	}
	return newPath, int(distance)
}

func AddPath(from string, to string, id string, weight int) error {

	_, err := query(
		`MATCH (n:Node {id: $from})
		MATCH (m:Node {id: $to}) 
		CREATE (n)-[:COSTS {id: $id, weight: $weight}]->(m)`,
		map[string]any{
			"from":   from,
			"to":     to,
			"id":     id,
			"weight": weight,
		})

	if err != nil {
		return err
	}
	return nil
}

func RemovePath(router1 string, router2 string) error {

	_, err := query(
		`MATCH (:Node {id: $router1})-[c:COSTS]-(:Node {id: $router2})
		DELETE c`,
		map[string]any{
			"router1": router1,
			"router2": router2,
		})

	if err != nil {
		return err
	}
	return nil

}
