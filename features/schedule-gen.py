"""
I want a table in this format:

Array:
          1,  2,  3,  4,  5
"08:00": A1, B1, C1, D1, E1
"09:00": F1, G1, A1, B1, C1 
"10:00": D1, E1, F1, G1, A1
"11:00": B1, C1, D1, E1, F1
"12:00": G1, A1, --, B1, C1 

{
    (1, "08:00"): "A1", (1, "09:00"): "F1", ...,
    (2, "08:00"): "B1", (2, "09:00"): "G1", ...,
    (3, "08:00"): "C1", (3, "09:00"): "A1", ...,
    ...
    ...
}

Give me python code to do this
"""

def theory(num: str, offset: int):
    days = [1, 2, 3, 4, 5]
    slots = ["08:00", "09:00", "10:00", "11:00", "12:00"]

    # Define the original schedule
    lookup = [
        ["A1", "B1", "C1", "D1", "E1"],
        ["F1", "G1", "A1", "B1", "C1"],
        ["D1", "E1", "F1", "G1", "TA1"],
        ["TB1", "TC1", "TD1", "TE1", "TF1"],
        ["TG1", "TAA1", "TBB1", "TCC1", "TD1"]
    ]

    keys = []
    for i in days:
        row = []
        for j in slots:
            if offset == 6:
                j = str(int(j[:2]) + offset) + ":00"
            row.append((i, j))
        keys.append(row)
    # print(keys)

    transposed_lookup = list(map(list, zip(*lookup)))
    #print(transposed_lookup)

    schedule = {}
    for ix in range(5):
        day, slot = keys[ix], transposed_lookup[ix]
        for id in range(5):
            schedule[day[id]] = slot[id].replace("1", num)

    # print("{")
    # [print(f"\t{key}: '{schedule[key]}',") for key in schedule]
    # print("}")
    #print(schedule)
    return schedule


def lab(num: int, offset: int):
    days = [1, 2, 3, 4, 5]
    slots = ["08:00", "09:50", "11:40"]

    # Define the original schedule
    lookup = [
        ["L1", "L7", "L13", "L19", "L25"],
        ["L3", "L9", "L15", "L21", "L27"],
        ["L5", "L11", "L17", "L23", "L29"],
    ]
    if num == 30:
        for r in range(len(lookup)):
            for c in range(len(lookup[r])):
                lookup[r][c] = "L" + str(int(lookup[r][c][1:]) + num)

    keys = []
    for i in days:
        row = []
        for j in slots:
            if offset == 6:
                j = str(int(j[:2]) + offset) + j[2:]
            row.append((i, j))
        keys.append(row)
    # print(keys)

    transposed_lookup = list(map(list, zip(*lookup)))
    #print(transposed_lookup)

    schedule = {}
    for ix in range(5):
        day, slot = keys[ix], transposed_lookup[ix]
        for id in range(3):
            schedule[day[id]] = slot[id]

    # print("{")
    # [print(f"\t{key}: '{schedule[key]}',") for key in schedule]
    # print("}")
    #print(schedule)
    return schedule

def merge_dicts(*dicts):
    result = {}
    for d in dicts:
        for key, value in d.items():
            if key in result:
                result[key].append(value)
            else:
                result[key] = [value]
    return result

# Morning schedule
TM = theory("1", 0)
# Evening schedule
TE = theory("2", 6)
# Morning schedule
LM = lab("", 0)
# Evening schedule
LE = lab(30, 6)

fin_dict = merge_dicts(TM,TE,LM,LE)
print(fin_dict)