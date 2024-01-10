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

def main(num: str, offset: int):
    days = [1, 2, 3, 4, 5]
    slots = ["08:00", "09:00", "10:00", "11:00", "12:00"]

    # Define the original schedule
    lookup = [
        ["A1", "B1", "C1", "D1", "E1"],
        ["F1", "G1", "A1", "B1", "C1"],
        ["D1", "E1", "F1", "G1", "A1"],
        ["B1", "C1", "D1", "E1", "F1"],
        ["G1", "A1", "--", "B1", "C1"]
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
    # print(transposed_lookup)

    schedule = {}
    for ix in range(5):
        day, slot = keys[ix], transposed_lookup[ix]
        for id in range(5):
            schedule[day[id]] = slot[id].replace("1", num)

    print("{")
    [print(f"\t{key}: '{schedule[key]}',") for key in schedule]
    print("}")

# Morning schedule
main("1", 0)
# Evening schedule
main("2", 6)
