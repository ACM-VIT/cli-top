from datetime import datetime, timedelta

def generate_schedule():
    days = 5  # Monday to Friday
    start_time = datetime.strptime("08:00", "%H:%M")
    end_time = datetime.strptime("12:00", "%H:%M")

    # Define the slots
    slots = ["A", "B", "C", "D", "E", "F", "G"]

    schedule_dict = {}

    for day in range(1, days + 1):
        current_time = start_time
        while current_time <= end_time:
            for slot in slots:
                key = (day, current_time.strftime("%H:%M"))
                value = f"{slot}{day}"
                schedule_dict[key] = value

                # Increment time by 1 hour
                current_time += timedelta(hours=1)

    return schedule_dict

# Example usage:
schedule = generate_schedule()
print(schedule)
