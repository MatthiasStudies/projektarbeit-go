
import os



STRESS_MULTI = "stress_multi.gotest"
STRESS_SINGLE ="stress_single.gotest"


NUM_STRUCTS = 5_000

t_multi = ""
t_single = ""

def write_single(line: str):
    global t_single
    t_single += line
    
def write_multi(line: str):
    global t_multi
    t_multi += line

def write_both(line: str):
    write_single(line)
    write_multi(line)




write_both("package main\n\n")

for i in range(NUM_STRUCTS):
    write_both(f"type Struct{i} struct {{}}\n")

    write_both(f"func (s *Struct{i}) Method() string {{\n")
    write_both(f'    return "Struct{i} Method called"\n')
    write_both("}\n\n")

write_both("type InterfaceStress1 interface {\n")
write_both("    Method() string\n")
write_both("}\n\n")
write_both("func main() {\n")

for i in range(NUM_STRUCTS):
    write_multi(f"    var s{i} InterfaceStress1 = &Struct{i}{{}}\n")
    write_multi(f'    _ = s{i}.Method()\n')

for i in range(NUM_STRUCTS):
    write_single(f"    var s{i} InterfaceStress1 = &Struct1{{}}\n")
    write_single(f'    _ = s{i}.Method()\n')

write_both("}\n")


with open(STRESS_MULTI, "w") as f:
    f.write(t_multi)

with open(STRESS_SINGLE, "w") as f:
    f.write(t_single)