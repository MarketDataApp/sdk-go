#!/usr/bin/env python3
import sys
import re
import os

# Central mapping of URLs to their corresponding title and sidebar position
URL_TO_INFO = {
    "https://www.marketdata.app/docs/api/markets/status": {"title": "Status", "sidebar_position": 1},
    "https://www.marketdata.app/docs/api/indices/candles": {"title": "Candles", "sidebar_position": 1},
    "https://www.marketdata.app/docs/api/indices/quotes": {"title": "Quotes", "sidebar_position": 2},
    "https://www.marketdata.app/docs/api/stocks/candles": {"title": "Candles", "sidebar_position": 1},
    "https://www.marketdata.app/docs/api/stocks/bulkcandles": {"title": "Bulk Candles", "sidebar_position": 2},
    "https://www.marketdata.app/docs/api/stocks/quotes": {"title": "Quotes", "sidebar_position": 3},
    "https://www.marketdata.app/docs/api/stocks/bulkquotes": {"title": "Bulk Quotes", "sidebar_position": 4},
    "https://www.marketdata.app/docs/api/stocks/earnings": {"title": "Earnings", "sidebar_position": 5},
    "https://www.marketdata.app/docs/api/stocks/news": {"title": "News", "sidebar_position": 6},
    "https://www.marketdata.app/docs/api/options/expirations": {"title": "Expirations", "sidebar_position": 1},
    "https://www.marketdata.app/docs/api/options/lookup": {"title": "Lookup", "sidebar_position": 2},
    "https://www.marketdata.app/docs/api/options/strikes": {"title": "Strikes", "sidebar_position": 3},
    "https://www.marketdata.app/docs/api/options/chain": {"title": "Option Chain", "sidebar_position": 4},
    "https://www.marketdata.app/docs/api/options/quotes": {"title": "Quotes", "sidebar_position": 5},
    "https://www.marketdata.app/docs/sdk/go/client": {"title": "Client", "sidebar_position": 3},

    # Add more mappings as needed
}

def clean_headings(content, pattern):
    # Split the content into lines for processing
    lines = content.split('\n')
    modified_lines = []
    
    for line in lines:
        # Check if the line matches the pattern
        if line.strip().startswith('#') and pattern in line:
            # Count the number of '#' characters at the start of the line
            hash_count = line.count('#', 0, line.find(' '))
            # Extract the last word from the line
            last_word = line.split()[-1]
            # Replace the original line with the modified version
            modified_line = '#' * hash_count + ' ' + last_word
            modified_lines.append(modified_line)
        else:
            # If the line does not match the pattern, keep it as is
            modified_lines.append(line)
    
    # Join the modified lines back into a single string
    modified_content = '\n'.join(modified_lines)
    return modified_content

def colapse_bullet_points(content):
    lines = content.split('\n')
    modified_lines = []
    i = 0
    while i < len(lines):
        # Check if the current line meets the criteria
        if lines[i].startswith('- ') and lines[i].count('`') == 2:
            # Check if the next line is blank and the one after starts with two spaces
            if i + 1 < len(lines) and lines[i + 1] == '' and i + 2 < len(lines) and lines[i + 2].startswith('  '):
                # Concatenate the current line with the description (third line), then append
                combined_line = lines[i] + ' ' + lines[i + 2].strip()
                modified_lines.append(combined_line)
                i += 3
                continue
        # Append the current line if it doesn't meet the criteria
        modified_lines.append(lines[i])
        i += 1
    # Join the modified lines back into a single string
    return '\n'.join(modified_lines)

def move_responses_to_top(file_content):
    # Find all unique response struct patterns
    response_struct_patterns = set(re.findall(r'<a name="([^"]*Response)(?=")"></a>', file_content))
    # Find all <a name= patterns that do not match the "Response" pattern
    non_response_patterns = set(re.findall(r'<a name="((?!Response).)*">', file_content))
    
    modified_content = file_content
    for pattern in response_struct_patterns:
        # Construct the exact start pattern for each response struct
        start_pattern = f'<a name="{pattern}"></a>'
        # Initialize end_pattern as None
        end_pattern = None
        # Find the closest non-response pattern after each response pattern
        for non_response in non_response_patterns:
            non_response_index = file_content.find(f'<a name="{non_response}"></a>')
            response_index = file_content.find(start_pattern)
            if non_response_index > response_index:
                end_pattern = f'<a name="{non_response}"></a>'
                break  # Break after finding the closest non-response pattern
        
        if end_pattern:
            # Call move_to_top for each response struct pattern with the found end_pattern
            modified_content = move_to_top(modified_content, start_pattern, end_pattern)
        else:
            # If no end_pattern is found, it means this response struct is the last section
            modified_content = move_to_top(modified_content, start_pattern, None)
    
    return modified_content

def move_to_top(file_content, start_pattern, end_pattern):
    lines = file_content.split('\n')
    
    sections = []
    start_index = None
    
    for i, line in enumerate(lines):
        if line.startswith(start_pattern):
            if start_index is not None:
                # Save the previous section if a new one starts before the old one ends
                sections.append((start_index, i))
            start_index = i  # Mark the start of a new section
        elif end_pattern is not None and line.startswith(end_pattern) and start_index is not None:
            # Mark the end of the current section and reset start_index if end_pattern is not None
            sections.append((start_index, i))
            start_index = None
    
    # If end_pattern is None, capture all the way to the end of the file from the start_pattern
    if start_index is not None:
        sections.append((start_index, len(lines)))
    
    # Extract sections and the rest of the content
    section_contents = [lines[start:end] for start, end in sections]
    rest_of_content = [line for i, line in enumerate(lines) if not any(start <= i < end for start, end in sections)]
    
    # Combine the extracted sections with the rest of the content, placing sections at the top
    modified_content = [line for section in section_contents for line in section] + rest_of_content
    
    return '\n'.join(modified_content)

def move_to_bottom(file_content, start_pattern, end_pattern):
    lines = file_content.split('\n')
    
    sections = []
    start_index = None
    
    for i, line in enumerate(lines):
        if line.startswith(start_pattern):
            if start_index is not None:
                # Save the previous section if a new one starts before the old one ends
                sections.append((start_index, i))
            start_index = i  # Mark the start of a new section
        elif line.startswith(end_pattern) and start_index is not None:
            # Mark the end of the current section and reset start_index
            sections.append((start_index, i))
            start_index = None
    
    # Check if the last section goes until the end of the file
    if start_index is not None:
        sections.append((start_index, len(lines)))
    
    # Extract sections and the rest of the content
    section_contents = [lines[start:end] for start, end in sections]
    rest_of_content = [line for i, line in enumerate(lines) if not any(start <= i < end for start, end in sections)]
    
    # Combine the rest of the content with the extracted sections
    modified_content = rest_of_content + [line for section in section_contents for line in section]
    
    return '\n'.join(modified_content)


def find_method_blocks_and_relocate(content):
    lines = content.split('\n')
    method_suffixes = [".Get", ".Raw", ".Packed"]
    method_blocks = {}
    type_definition_found = False
    type_end_loop = False
    type_name = ""
    type_end_line = 0
    first_method_line_number = None  # Initialize variable to track the first method line number

    for i, line in enumerate(lines):
        stripped_line = line.strip()

        if not type_definition_found and stripped_line.startswith('<a') and "Request" in stripped_line:
            type_definition_found = True
            type_name = stripped_line.split('"')[1]
            continue

        if type_definition_found and not type_end_loop:
            for j in range(i+1, len(lines)):
                if "## type" in lines[j]:
                    type_end_line = j
                    break
            else:
                type_end_line = len(lines) - 1
            type_end_loop = True

        # Check if the line is a method of the current type_name
        if type_definition_found and stripped_line.startswith(f'<a name="{type_name}.'):
            start_line_number = i
            for j in range(i+1, len(lines)):
                if lines[j].strip().startswith('<a'):
                    end_line_number = j - 1
                    break
            # Update first_method_line_number if this is the first method found
            if first_method_line_number is None or start_line_number < first_method_line_number:
                first_method_line_number = start_line_number

            # Check if the method matches one of the specific suffixes for relocation
            for method_suffix in method_suffixes:
                if f'<a name="{type_name}{method_suffix}"></a>' in stripped_line:
                    method_blocks[method_suffix] = {
                        'type_name': type_name,
                        'start_line_number': start_line_number,
                        'end_line_number': end_line_number,
                        'content': '\n'.join(lines[start_line_number:end_line_number+1])
                    }
                    break

    # Sort the method blocks in the specified order
    sorted_methods = sorted(method_blocks.items(), key=lambda x: [".Get", ".Packed", ".Raw"].index(x[0]))

    if type_definition_found:
        # Build the header for execution methods
        header_to_insert = f"## {type_name} Execution Methods"
        lines.insert(type_end_line, header_to_insert)

    # Copy and paste the method blocks after the type definition
    for _, method_info in sorted_methods:
        lines.insert(type_end_line + 1, method_info['content'])
        type_end_line += len(method_info['content'].split('\n'))

    # Delete the original method blocks in reverse order to maintain correct line numbers
    for _, method_info in sorted(method_blocks.items(), reverse=True):
        del lines[method_info['start_line_number']:method_info['end_line_number']+1]

    # Insert the header for setter methods before the first method
    if type_definition_found and first_method_line_number is not None:
        setter_methods_header = f"## {type_name} Setter Methods"
        lines.insert(first_method_line_number - 1, setter_methods_header)

    return '\n'.join(lines)

def remove_code_block_delimiters(content, section_title):
    """
    Find the section by title and remove the first two occurrences of code block delimiters (```) after the section title.
    """
    lines = content.split('\n')
    new_lines = []
    in_section = False
    code_block_delimiters_removed = 0

    for line in lines:
        if line.strip() == section_title:
            in_section = True
        elif in_section and line.strip() == '```' and code_block_delimiters_removed < 2:
            # Skip the first two occurrences of code block delimiters after the section title
            code_block_delimiters_removed += 1
            continue
        new_lines.append(line)
        # If we've removed two code block delimiters, we can stop checking further lines
        if code_block_delimiters_removed == 2:
            in_section = False

    return '\n'.join(new_lines)


def remove_first_sentence(content):
    """Remove the first sentence from the content, searching from top to bottom for a line that begins with 'Package client'."""
    # Split the content into lines
    lines = content.split('\n')
    new_lines = []
    found = False

    for line in lines:
        if not found and line.startswith('Package'):
            # Find the first period in the line
            period_index = line.find('.')
            if period_index != -1:
                # Remove up to and including the first period
                new_line = line[period_index + 1:].lstrip()
                if new_line:  # If there's any content left in the line, add it
                    new_lines.append(new_line)
                found = True  # Mark as found to stop modifying lines
                continue
        new_lines.append(line)

    return '\n'.join(new_lines)

def add_anchor_tags(content, sections_to_process):
    """Process the parameters block of text as specified, including 'Setter Methods' and 'Execution Methods'. 
    Insert <a href="#"> before and </a> after lines that start with a dash (-)."""
    lines = content.split('\n')
    processed_lines = []
    processing = False
    found_first_dash = False  # Variable to track the first dash

    for line in lines:
        if (match := line.strip()) in sections_to_process:
            processing = True
            matched_string = match  # Keep track of which string matched
            found_first_dash = False  # Reset for each new section
            processed_lines.append(line)
            continue
        if processing:
            if not found_first_dash:
                if line.strip() == '':
                    processed_lines.append(line)
                    continue
                elif not line.lstrip().startswith('-'):
                    processed_lines.append(line)
                    continue
                else:
                    found_first_dash = True  # Found the first dash, start processing lines
            if line.startswith('-'):
                if matched_string != '#### Generated By':
                    previous_lines = lines[:lines.index(line)]
                    type_word = ""
                    for prev_line in reversed(previous_lines):
                        if prev_line.strip().startswith("type "):
                            type_word = prev_line.split()[1]
                            break
                    anchor = line[line.find('`')+1:line.find('(')].strip()
                    # Insert <a href="#"> before and </a> after the line
                    processed_lines.append('- ' + f'<a href="#{type_word}.{anchor}">' +'`' + line[3:] + '</a>')
                else:
                    anchor = line[line.find('`')+1:line.find('(')].strip()
                    # Insert <a href="#"> before and </a> after the line
                    processed_lines.append('- ' + f'<a href="#{anchor}">' +'`' + line[3:] + '</a>')
            elif line.strip() == '' or line.startswith('  '):
                processed_lines.append(line)
                continue
            else:
                # Stop processing if the line does not start with a dash
                processing = False
                processed_lines.append(line)
        else:
            processed_lines.append(line)

    return '\n'.join(processed_lines)

def move_all_struct_definitions(content):
    """Move all struct definition blocks right after their type documentation."""
    import re

    # Pattern to find struct type documentation
    struct_doc_pattern = re.compile(r"## type (\w+)")
    # Pattern to find struct definition blocks
    struct_def_pattern_template = r"(```go\s+type {}\s.*?```)"
    
    # Find all struct names from the documentation
    struct_names = struct_doc_pattern.findall(content)

    for struct_name in struct_names:
        # Create a pattern for the current struct definition
        struct_def_pattern = re.compile(struct_def_pattern_template.format(struct_name), re.DOTALL)
        
        # Find the struct definition block
        struct_def_match = struct_def_pattern.search(content)
        if not struct_def_match:
            continue  # Skip if struct definition block not found

        struct_def_block = struct_def_match.group(1)

        # Remove the original struct definition block from the content
        content = struct_def_pattern.sub('', content, count=1)

        # Insert the struct definition block right after the struct type documentation
        content = re.sub(
            rf"(## type {struct_name}\s*\n)",
            r'\1' + struct_def_block + '\n\n',
            content,
            count=1
        )

    return content

def correct_escaping_in_links(content):
    """Correct escaping in markdown links that start with [/v1 and enclose URL text in backticks if it contains { or }."""
    # Regular expression to match the pattern
    pattern = re.compile(r'(\[/v1[^\]]+\])')
    
    def remove_escapes_and_check_braces(match):
        # Remove backslashes from the matched string
        cleaned_match = match.group(0).replace('\\', '')
        # Check if the URL text contains { or }
        if '{' in cleaned_match or '}' in cleaned_match:
            # Enclose the URL text in backticks
            return cleaned_match[0] + '`' + cleaned_match[1:-1] + '`' + cleaned_match[-1]
        return cleaned_match
    
    # Replace all occurrences of the pattern with their escaped characters removed
    # and check for { or } to enclose in backticks
    corrected_content = re.sub(pattern, remove_escapes_and_check_braces, content)
    return corrected_content

def remove_index_block(content):
    """Remove the index block from the markdown content, starting after the first new line after '## Index'."""
    lines = content.split('\n')
    new_lines = []
    in_index_block = False
    past_first_new_line = False  # Track if we're past the first new line after '## Index'

    for line in lines:
        if line.startswith('## Index'):
            in_index_block = True
            continue
        if in_index_block and not past_first_new_line:
            if line.strip() == '':
                past_first_new_line = True  # We're past the first new line, start processing next lines
            continue  # Skip until we're past the first new line
        if in_index_block:
            if line.strip() == '' or not line.lstrip().startswith('-'):
                in_index_block = False  # End of index block
                continue
        else:
            new_lines.append(line)

    return '\n'.join(new_lines)

def process_header_blocks(content, blocks_to_process):
    """Process the parameters block of text as specified, including 'Setter Methods' and 'Execution Methods'."""
    lines = content.split('\n')
    processed_lines = []
    processing = False
    found_first_dash = False  # New variable to track the first dash

    for line in lines:
        if line.strip() in blocks_to_process:
            processing = True
            found_first_dash = False  # Reset for each new section
            processed_lines.append(line)
            continue
        if processing:
            if not found_first_dash:
                if line.strip() == '':
                    processed_lines.append(line)
                    continue
                elif not line.startswith('-'):
                    processed_lines.append(line)
                    continue
                else:
                    found_first_dash = True  # Found the first dash, start processing lines
            if line.startswith('-'):
                # Step 1: Add a backtick after the dash and before the first colon
                line = line.replace('- ', '- `', 1)
                # Step 2: Replace the first colon with two new lines and two spaces
                line = line.replace(':', '`\n\n ', 1)
                # Step 3: Now, remove escape characters only between the backticks we've just added
                parts = line.split('`')
                if len(parts) > 2:  # Ensure there are backticks to process
                    parts[1] = parts[1].replace('\\', '')  # Remove escape characters only in the part between backticks
                    line = '`'.join(parts)
                # Step 4: Add an additional new line at the end
                line += '\n'
                processed_lines.append(line)
            else:
                # Stop processing if the line does not start with a dash
                processing = False
                processed_lines.append(line)
        else:
            processed_lines.append(line)

    return '\n'.join(processed_lines)

def remove_output_blocks(content):
    """Remove blocks of text starting with '// Output:' and ending with '```', including the start line but not the end line."""
    output_pattern = re.compile(r'// Output:.*?```', re.DOTALL)
    # Replace the found blocks with just '```' to keep the ending line
    cleaned_content = re.sub(output_pattern, '```', content)
    return cleaned_content

def add_tabs_tags(content):
    """Add opening and closing <Tabs> tags around groups of <TabItem> tags, considering blank lines."""
    lines = content.split('\n')
    new_lines = []
    in_tab_group = False

    def is_next_non_blank_line_tabitem(start_index, direction):
        """Check if the next non-blank line in the given direction (1 for forward, -1 for backward) is a <TabItem> line."""
        index = start_index + direction
        while 0 <= index < len(lines) and lines[index].strip() == '':
            index += direction
        if 0 <= index < len(lines):
            return lines[index].strip().startswith('<TabItem') if direction == 1 else lines[index].strip().endswith('</TabItem>')
        return False

    for i, line in enumerate(lines):
        trimmed_line = line.strip()
        # Check for opening <TabItem> without preceding closing </TabItem>
        if trimmed_line.startswith('<TabItem') and not is_next_non_blank_line_tabitem(i, -1):
            if not in_tab_group:
                new_lines.append('<Tabs>')
                in_tab_group = True
        new_lines.append(line)
        # Check for closing </TabItem> without following opening <TabItem>
        if trimmed_line.endswith('</TabItem>') and not is_next_non_blank_line_tabitem(i, 1):
            if in_tab_group:
                new_lines.append('</Tabs>')
                in_tab_group = False

    return '\n'.join(new_lines)

def convert_details_to_tabitem(content):
    """Convert <details> tags to <TabItem> with dynamic attributes based on the summary content, and remove the trailing <p>."""
    detail_pattern = re.compile(
        r'<details><summary>(.*?)</summary>\n<p>',
        re.DOTALL
    )
    return re.sub(detail_pattern, r'<TabItem value="\1" label="\1">\n', content)
def read_file_content(file_path):
    """Read and return the content of the file."""
    try:
        with open(file_path, 'r') as file:
            return file.read()
    except FileNotFoundError:
        print(f"Error: The file {file_path} was not found.")
        return None

def write_file_content(file_path, content):
    """Write the given content to the file."""
    with open(file_path, 'w') as file:
        file.write(content)

def remove_pattern(content, patterns):
    """Remove all occurrences of the patterns from the content."""
    for pattern in patterns:
        pattern_str = r'\n?'.join([re.escape(part) for part in pattern])
        pattern_re = re.compile(pattern_str, re.DOTALL)
        content = re.sub(pattern_re, '', content, 1)
    return content

def replace_pattern(content, replacements):
    """Replace occurrences based on a dictionary of find-and-replace pairs."""
    for find, replace in replacements.items():
        content = content.replace(find, replace)
    return content

def process_file(file_path):
    """Process the file to remove specified patterns, replace specified strings, and convert <details> to <TabItem>, including removing the trailing <p>."""
    content = read_file_content(file_path)
    if content is not None:
        patterns_to_delete = [
            ['', '```go', 'import "."', '```', ''],
            ['# client'],
            ['# models'],
            ['# dates'],
            ['Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)'],
            ['<!-- Code generated by gomarkdoc. DO NOT EDIT -->']
        ]
        content = remove_pattern(content, patterns_to_delete)

        replacements = {
        '\n### Notes': '\n#### Notes',
        '\n#### Notes': '\n#### Notes',
        '#### Output': '#### Output',
        '#### Parameters': '#### Parameters',
        '#### Returns': '#### Returns',
        '### Making Requests': '## Making Requests',
        '### Setter Methods': '#### Setter Methods',
        '### Execution Methods': '#### Execution Methods',
        '### Methods': '#### Methods',
        '### Generated By': '#### Generated By',
        '</p>\n</details>': '</TabItem>'  # Generate closing MDX tabs
        }
        content = replace_pattern(content, replacements)

        content = remove_first_sentence(content)
        content = convert_details_to_tabitem(content)  # Convert <details> to <TabItem> and remove trailing <p>
        content = add_tabs_tags(content)  # Add <Tabs> and </Tabs> tags
        content = remove_output_blocks(content)  # Remove output blocks

        blocks_to_process = ['#### Parameters','#### Returns', '#### Setter Methods', '#### Execution Methods', '#### Methods', '#### Generated By']    
        content = process_header_blocks(content, blocks_to_process)  # Process header blocks

        content = remove_index_block(content)  # Remove index block
        content = correct_escaping_in_links(content)  # Correct escaping in links
        content = move_all_struct_definitions(content) # Move all struct definitions

        sections_to_process = ['#### Setter Methods', '#### Execution Methods', '#### Methods', '#### Generated By']
        content = add_anchor_tags(content, sections_to_process)

        content = remove_code_block_delimiters(content, "## Making Requests")
        content = find_method_blocks_and_relocate(content)
        content = move_responses_to_top(content)
        #content = move_to_bottom(content, '<a name="Candle','<a' )
        content = move_to_bottom(content, '<a name="By','<a' )
        # content = colapse_bullet_points(content)

        replacements = {
        '## type': '##',
        }
        content = replace_pattern(content, replacements)

        content = clean_headings(content, 'func')

        # content = add_anchor_tags_to_generated_by(content)  # Add anchor tags to 'Generated By' sections

        write_file_content(file_path, content)
        print(f"File {file_path} has been processed.")

def combine_files_into_mdx(file_paths):
    """Prepare the combined content of multiple files."""
    combined_content = ""
    for file_path in file_paths:
        try:
            with open(file_path, 'r') as file:
                combined_content += file.read() + "\n\n"
        except Exception as e:
            print(f"Error reading file {file_path}: {e}")
            return None, None

    global URL_TO_INFO
    urls_found = re.findall(r'https?://[^\s)>]+', combined_content)
    urls_to_process = [url for url in urls_found if url in URL_TO_INFO]

    if not urls_to_process:
        print(f"Error: No URLs matching URL_TO_INFO found in the combined content.\nURLs found: {urls_found}\nURLs expected but not found: {[url for url in URL_TO_INFO.keys() if url not in urls_found]}")
        sys.exit(1)  # Exit the script if no matching URLs are found

    for url in urls_to_process:
        info = URL_TO_INFO.get(url)
        if info:
            header_text = f"---\ntitle: {info['title']}\nsidebar_position: {info['sidebar_position']}\n---\n\n"
            combined_content = header_text + combined_content
            # Use consistent logic for both /api/ and /sdk/
            if "/sdk/" in url:
                base_name = "go"
                path_after_segment = url.split("/go/")[-1]
            elif "/api/" in url:
                base_name = "go"
                path_after_segment = url.split("/api/")[-1]
            else:
                continue  # Skip if URL does not contain /api/ or /sdk/
            output_filename = f"{base_name}/{path_after_segment}.mdx"
            break  # Exit the loop after processing the first matching URL

    if "<Tabs>" in combined_content:
        import_statements = "import Tabs from \"@theme/Tabs\";\nimport TabItem from \"@theme/TabItem\";\n\n"
        combined_content = combined_content.replace("---\n\n", "---\n\n" + import_statements, 1)

    return combined_content, output_filename

def save_combined_content(combined_content, output_filename):
    """Save the combined content to a file, creating necessary directories."""
    if not combined_content or not output_filename:
        print("Error: Missing combined content or output filename.")
        return

    output_directory = os.path.dirname(output_filename)
    if output_directory and not os.path.exists(output_directory):
        try:
            os.makedirs(output_directory)
        except Exception as e:
            print(f"Error creating directory {output_directory}: {e}")
            return

    try:
        with open(output_filename, 'w') as output_file:
            output_file.write(combined_content)
        print(f"Combined MDX file created at {output_filename}")
    except Exception as e:
        print(f"Error writing to file {output_filename}: {e}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: ./process_markdown.py <file_path> [<file_path> ...]")
        sys.exit(1)
    
    for file_path in sys.argv[1:]:
        process_file(file_path)    
    # Combine all processed files into a single .mdx file
    process_file_paths = sys.argv[1:]  # Assuming these are the paths of processed files
    combined_content, output_filename = combine_files_into_mdx(process_file_paths)
    if combined_content is not None and output_filename is not None:
        save_combined_content(combined_content, output_filename)