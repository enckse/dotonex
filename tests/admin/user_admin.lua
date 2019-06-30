object = network:Define("1", "testid")
object.Macs = {VALID_MAC}
object:Assigned(DEV)
network:Own("owned", {"bbbbaaaabbbb"})
