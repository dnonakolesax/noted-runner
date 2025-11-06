import sys
import subprocess
import os
import json

# do not forget to install goimports!

filename = sys.argv[1]
basepath = sys.argv[2]

signatures_old = {}
signatures = {}
try:
    with open(basepath + "/signatures.json", "r") as infile:
        signatures_old = json.load(infile)
except:
    pass        

def parse(filename, basepath):
    pureFilename = filename
    filename = basepath + "/" + filename
    needFunc = False
    figureStack = []
    writeFile = open(filename + ".go", "w")
    seps = ['', '\n', '\t']
    funcs = []
    export = []

    with open(basepath + "/base") as f:
        for line in f:
            writeFile.write(line)


    with open(filename) as f:
        lines = f.readlines()

        for line in lines:
            i = 0

            if len(figureStack) == 0:
                while i < len(line) and line[i] in seps:
                    i += 1
                isFunc = line[i:].split()
                if len(isFunc) > 0 and (len(isFunc[0].split('(')) > 0) and isFunc[0] != "func":
                    funName = isFunc[0].split('(')[0]
                    if funName in signatures_old and funName not in signatures:
                        needFunc = True
                        export.append(funName + ' := funcsMap["' + funName + '"].(' + signatures_old[funName] + ')\n')
                        export.append(line)                
                    elif funName in signatures:
                        needFunc = True
                        export.append(line)     
                    else:
                        export.append(line)
                elif len(isFunc) > 0 and isFunc[0] == "func":    
                    writeFile.write(line)
                    funcName = isFunc[1].split('(')[0]
                    #print(funcName)
                    paramTypes = isFunc[2].split(')')
                    signatures[funcName] = 'func(' + paramTypes[0] + ')'
                    print(funcName, signatures[funcName])
                    funcs.append(funcName)
                    i += 4
                    while i < len(line) and line[i] != '{':
                        i += 1
                    figureStack.append('{') 
                    i += 1    

                    while i < len(line) and len(figureStack) > 0:
                        if line[i] == '{':
                            figureStack.append('{')
                        elif line[i] == '}':
                            figureStack.pop()           
                        i += 1    
                else:
                    if line != '\n':
                        print("export append")
                        export.append(line)    

            else:
                writeFile.write(line)  
                while i < len(line) and len(figureStack) > 0:
                    if line[i] == '{':
                        figureStack.append('{')
                    elif line[i] == '}':
                        figureStack.pop()              
                    i += 1    

    if len(funcs) == 0 and needFunc == False:
        writeFile.write('\n func Export_' + pureFilename + '(_ *map[string]any, _ *map[string]any) {')
    else:
        writeFile.write('\n func Export_' + pureFilename + '(_ *map[string]any, funcMap *map[string]any) {')
        writeFile.write('funcsMap := *funcMap \n')


    for exportStr in export:
        print("write")
        writeFile.write(exportStr)

    for func in funcs:
        writeFile.write('\tfuncsMap["' + func + '"] = ' + func + '\n')

    writeFile.write('}')
    writeFile.close()

    cmd = 'goimports -w ' + filename + '.go'
    #print(cmd)
    os.system(cmd)

    cmd = 'go build -buildmode=plugin -o ' + filename + '.so ' + '' + filename + '.go'
    #print(cmd)
    os.system(cmd)
    #print(export)

parse(filename, basepath)

signatures = {**signatures_old, **signatures}
with open(basepath + "/signatures.json", "w") as outfile:
    outfile.write(json.dumps(signatures))