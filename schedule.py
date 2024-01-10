# Define the days and slots
days = [1, 2, 3, 4, 5]
slots = ["08:00", "09:00", "10:00", "11:00", "12:00"]
letters = ["A1", "F1", "D1", "B1", "G1"]

# Initialize an empty dictionary
schedule = {}

# Iterate over the days and slots
for i, day in enumerate(days):
   schedule[(day, slots[i])] = letters[i]

# Print the dictionary
print(schedule)
