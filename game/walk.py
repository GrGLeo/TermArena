import os

# Define the starting directory for the recursive search ('.' means the current directory)
root_directory = "."
# Define the file extension to search for
extension = ".rs"

print(f"Searching for files with extension '{extension}' recursively starting from '{root_directory}'...")
print("-" * 50) # Print a separator line

# Use os.walk to traverse the directory tree starting from root_directory
# os.walk yields a tuple for each directory it visits: (current_folder_path, list_of_subdirectories, list_of_files)
for folder_path, sub_folders, file_names in os.walk(root_directory):
    # Iterate through the list of files in the current folder
    for file_name in file_names:
        # Check if the file name ends with the specified extension
        if file_name.endswith(extension):
            # Construct the full path to the file
            file_path = os.path.join(folder_path, file_name)

            # Print a header indicating the file path
            print(f"--- File Path: {file_path} ---")
            try:
                # Open and read the content of the file with UTF-8 encoding
                with open(file_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                    # Print the content of the file
                    print(content)
                # Print a newline for better separation between the content of different files
                print("\n")
            except Exception as e:
                # If an error occurs while reading the file, print an error message
                print(f"Error reading file {file_path}: {e}")

print("-" * 50) # Print a separator line at the end
print("Finished processing files.")

