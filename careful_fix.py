#!/usr/bin/env python3
"""
Carefully add .Run() only where needed.
"""

with open('command_test.go', 'r') as f:
    lines = f.readlines()

# First pass: identify which line ranges are `result := run.Command...` statements
result_statements = []
i = 0
while i < len(lines):
    line = lines[i]
    if '\tresult := run.Command' in line or '\tresult := run.Quick' in line or '\tresult := run.WithInput' in line:
        start = i
        # Find the end of the statement (next line that doesn't start with tabs or is empty)
        end = i + 1
        while end < len(lines):
            next_line = lines[end]
            if next_line.strip() == '':
                # Empty line means end of statement
                break
            if next_line.startswith('\t\t'):  # Continuation
                end += 1
            else:
                break
        
        # Find the last non-empty line in this range
        last_code_line = end - 1
        while last_code_line > start and lines[last_code_line].strip() == '':
            last_code_line -= 1
        
        result_statements.append((start, last_code_line))
        i = end
    else:
        i += 1

# Second pass: for each result statement, check if it needs .Run()
for start, last_code_line in result_statements:
    last_line = lines[last_code_line]
    
    # Skip if it already has .Run() or is Quick/WithInput (which don't need it)
    if '.Run()' in last_line:
        continue
    if 'run.Quick(' in lines[start]:
        continue  
    if 'run.WithInput(' in lines[start]:
        continue
    
    # Add .Run() if the line ends with ), ) or )\n
    if last_line.rstrip().endswith(')'):
        # Remove any incorrect .Run() that might have been added to wrong places
        if 'assertion.' in last_line:
            continue  # Don't touch assertion lines
        
        lines[last_code_line] = last_line.rstrip() + '.Run()\n'

# Third pass: remove .Run() from assertion lines (if any were added by mistake)
for i in range(len(lines)):
    if 'assertion.' in lines[i] and ').Run()' in lines[i]:
        lines[i] = lines[i].replace(').Run()', ')')

with open('command_test.go', 'w') as f:
    f.writelines(lines)

print("Fixed carefully!")
