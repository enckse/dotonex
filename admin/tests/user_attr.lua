object = network:Define("2", "testattr")
object.Macs = {VALID_MAC}
object.Make = "TEST"

obj2 = network:Define("3", "testattr3")
obj2.Macs = {"ffffffeeeeee"}
obj2.Model = "testtest"
obj2.Verbose = "extension,information test"

network:Assign(DEV, {object, obj2})
