import random

print("=== [TEST 01] QuickSort Algorithm Execution ===")
print("Action: Sorting 1,000 random integers...")

def quicksort(arr):
    if len(arr) <= 1:
        return arr
    pivot = arr[len(arr) // 2]
    left = [x for x in arr if x < pivot]
    middle = [x for x in arr if x == pivot]
    right = [x for x in arr if x > pivot]
    return quicksort(left) + middle + quicksort(right)

data = [random.randint(1, 10000) for _ in range(1000)]
sorted_data = quicksort(data)
print(f"Result: SUCCESS - Sorted array first 5 items: {sorted_data[:5]}")
print("=== END TEST 01 ===")
