#!/usr/bin/python
# -*- coding: UTF-8 -*-

import argparse
import base64
import codecs
import json
import os
import re

import get_workspace # 根据指定环境引入

EXTERFUNC = """extern "C" void __gcov_flush();\nstatic void catch_function(int signal) {__gcov_flush();}\n"""
SIGHANDLEFUNC = """if (signal(28, catch_function) == SIG_ERR){fputs("An error occurred while setting a signal handler.", stderr);return -1;}\n"""


def create_bash(task_info):
    REPO = os.getenv('REPO', None)
    print("REPO:", REPO)
    REPO_BRANCH = os.getenv('REPO_BRANCH', None)
    print("REPO_BRANCH:", REPO_BRANCH)
    REPO_COMMIT = os.getenv('REPO_COMMIT', None)
    print("REPO_COMMIT:", REPO_COMMIT)
    WORKSPACE = os.getenv('WORKSPACE', None)
    print("WORKSPACE:", WORKSPACE)
    CODEDOG_LANGUAGE = "c++"
    PROCESS_NAME = os.getenv('PROCESS_NAME', None)
    print("PROCESS_NAME:", PROCESS_NAME)
    BUILD_NUMBER = os.getenv('BUILD_NUMBER', None)
    print("BUILD_NUMBER:", BUILD_NUMBER)
    COMMIT_AUTHOR = os.getenv('COMMIT_AUTHOR', None)
    print("COMMIT_AUTHOR:", COMMIT_AUTHOR)
    JOB_ID = os.getenv('JOB_ID', None)
    print("JOB_ID:", JOB_ID)
    bash_str = "bash /usr/local/services/coverage_report/add_task.sh -repo " + REPO + " -branch " + REPO_BRANCH + " -commit " + REPO_COMMIT + " -data_dir " + WORKSPACE + " -dev_language " + CODEDOG_LANGUAGE + " -process_name " + PROCESS_NAME + " -build_no " + BUILD_NUMBER + " -committer " + COMMIT_AUTHOR + " --job_id " + JOB_ID
    if task_info is not None:
        bash_str = bash_str + " -info '%s'" % task_info
    return 'system("' + bash_str + '");\n'


def modify_main(main_path, task_info):
    update_bash = create_bash(task_info)
    work_space = get_workspace()
    main_path = os.path.join(work_space, main_path)
    print("main_path:", main_path)
    main_path_tmp = main_path + ".tmp"
    with codecs.open(main_path, encoding='utf-8', mode='r', errors='ignore') as f_read, open(main_path_tmp,
                                                                                             'w+') as f_write:
        lines = f_read.readlines()
        # print(lines)
        f_write.write("#include <signal.h>\n")
        f_write.write("#include <stdlib.h>\n")
        flag = 0
        for line_index in range(0, len(lines)):
            if flag == 1:
                f_write.write(lines[line_index])
                f_write.write(update_bash)
                f_write.write(SIGHANDLEFUNC)
                flag = 0
            # print(line)
            elif re.search(r'.+\smain\(.*\)', lines[line_index]):
                print("get main func")
                f_write.write(EXTERFUNC)
                f_write.write(lines[line_index])
                # 如果下一行是左括号，就在下一行后面加
                # 如果不是，就在当前行后面加
                if re.search(r'.*{.*', lines[line_index + 1]):
                    print("get {  at the next line of main func")
                    flag = 1
                else:
                    print("{ is at the same line of main func")
                    f_write.write(update_bash)
                    f_write.write(SIGHANDLEFUNC)
            else:
                f_write.write(lines[line_index])

    os.remove(main_path)
    os.rename(main_path_tmp, main_path)
    return


def modify_spp_handle_init(main_path, task_info):
    update_bash = create_bash(task_info)
    # main文件首行添加 #include <signal.h>
    print("change ssp_handle_init")
    work_space = get_workspace()
    main_path = os.path.join(work_space, main_path)
    print("spp_handle_init_path:", main_path)
    main_path_tmp = main_path + ".tmp"
    with codecs.open(main_path, encoding='utf-8', mode='r', errors='ignore') as f_read, open(main_path_tmp,
                                                                                             'w+') as f_write:
        lines = f_read.readlines()
        # print(lines)
        f_write.write("#include <signal.h>\n")
        # flag 标记是否下一行需要加插桩内容
        flag = 0
        init_func = 1
        for line_index in range(0, len(lines)):
            if flag == 1 and init_func:
                f_write.write(lines[line_index])
                f_write.write(update_bash)
                f_write.write(SIGHANDLEFUNC)
                init_func = 0
            # print(line)
            elif re.search(r'^extern "C".+ spp_handle_init.*', lines[line_index]):
                print("get spp init handle func")
                f_write.write(EXTERFUNC)
                f_write.write(lines[line_index])
            elif re.search(r'.+void spp_handle_fini.*', lines[line_index]):
                f_write.write(lines[line_index])
                init_func = 0
            elif re.search(r'.+SERVER_TYPE_WORKER.*', lines[line_index]) and init_func:
                # 如果下一行是左括号，就在下一行后面加
                # 如果不是，就在当前行后面加
                f_write.write(lines[line_index])
                if re.search(r'.*{.*', lines[line_index + 1]):
                    print("get {  at the next line of main func")
                    flag = 1
                else:
                    print("{ is at the same line of the block")
                    f_write.write(update_bash)
                    f_write.write(SIGHANDLEFUNC)
            else:
                # print(11111)
                f_write.write(lines[line_index])

    os.remove(main_path)
    os.rename(main_path_tmp, main_path)
    return


def build_flags(s, flags):
    flag = re.match(r'^(CFLAGS|CXXFLAGS|MYCXXFLAGS|MYCFLAGS|XXFLAGS)', s, re.I)
    if flag:
        f = flag.group(0)
        if f not in flags:
            flags.append(f)
            return f
        else:
            return None
    else:
        return None


def modify_makefile(makefile_path):
    # makefile 文件里面加入编译参数 -ftest-coverage -fprofile-arcs
    makefile_path_tmp = makefile_path + '.tmp'
    with codecs.open(makefile_path, encoding='utf-8', mode='rb', errors='ignore') as fread, open(makefile_path_tmp,
                                                                                                 'w',
                                                                                                 encoding="utf-8") as fwrite:
        lines = fread.readlines()
        flags = []
        lib_flag = ''
        for line in lines:
            cflags = build_flags(line, flags)
            if cflags != None:
                fwrite.write(line)
                cflags += " += -ftest-coverage -fprofile-arcs\n"
                fwrite.write(cflags)
            # 兼容动图链接lib库需要 -lgcov 参数
            elif re.search(r'^(MYLIB)', line) != None and lib_flag != 'done':
                lib_flag = re.match(r'^(MYLIB)', line, re.I).group(0)
                fwrite.write(line)
            elif lib_flag != '' and lib_flag != 'done':
                if re.search(r'^\s+.*', line) == None:
                    fwrite.write(lib_flag + ' += -lgcov\n')
                    lib_flag = 'done'
                    fwrite.write(line)
                else:
                    fwrite.write(line)
            else:
                fwrite.write(line)
        # print(flags)
    os.remove(makefile_path)
    os.rename(makefile_path_tmp, makefile_path)
    return


## 处理 cmakelists
def modify_cmakelists(cmake_path):
    # makefile 文件里面加入编译参数 -ftest-coverage -fprofile-arcs
    work_space = get_workspace()
    cmake_path = os.path.join(work_space, cmake_path)
    print("makefile_path:", cmake_path)
    cmake_path_tmp = cmake_path + '.tmp'
    with codecs.open(cmake_path, encoding='utf-8', mode='r', errors='ignore') as fread, open(cmake_path_tmp,
                                                                                             'w') as fwrite:
        lines = fread.readlines()
        flag = 0
        for line in lines:
            #  -*-  覆盖率参数添加在 project 后面  -*-
            # if re.search(r"^project.*", line, re.I) != None and flag == 0:
            #     fwrite.write(line)
            #     fwrite.write('SET(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fprofile-arcs -ftest-coverage")\n')
            #     fwrite.write('SET(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fprofile-arcs -ftest-coverage")\n')
            #     flag = 1
            # else:
            #     fwrite.write(line)
            #  -*-  覆盖率参数添加在 project 后面  -*-

            fwrite.write(line)
        fwrite.write('\n')
        fwrite.write('SET(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fprofile-arcs -ftest-coverage")\n')
        fwrite.write('SET(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fprofile-arcs -ftest-coverage")\n')
    os.remove(cmake_path)
    os.rename(cmake_path_tmp, cmake_path)
    return


make_file_list = []


def get_makefile_list(work_space):
    for root, dirs, files in os.walk(work_space):
        for file in files:
            if re.match(r'makefile.*', file, re.I):
                # print(os.path.join(root, file))
                make_file_list.append(os.path.join(root, file))


cmake_list = []


def get_cmake_list(work_space):
    for root, dirs, files in os.walk(work_space):
        for file in files:
            if re.match(r'cmakelists.*', file, re.I):
                # print(os.path.join(root, file))
                cmake_list.append(os.path.join(root, file))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='')
    parser.add_argument('--entrance_type', required=True,
                        help='main or spp')
    parser.add_argument('--main_path', required=True,
                        help='path of main')
    parser.add_argument('--makefile_list', nargs='+',
                        help='path of makefile')
    parser.add_argument('--filter_list', nargs='+',
                        help='path of filter dirs')
    parser.add_argument('--accumulate', action="store_true", help='whether accumulate history')
    args = parser.parse_args()

    # print(args.main_path)
    # print(args.makefile_path)
    print("entrance type:")
    print(args.entrance_type)
    print("main_path")
    print(args.main_path)
    print("makefile_list")
    print(type(args.makefile_list))
    print(args.makefile_list)
    print("filter_list")
    print(args.filter_list)
    print("accumulate")
    print(args.accumulate)

    # 添加任务参数
    task_info_j = {}
    if args.filter_list is not None:
        for i in args.filter_list:
            if i == "":
                args.filter_list.remove(i)
        if args.filter_list == [""]:
            args.filter_list = None
        else:
            task_info_j["filter"] = args.filter_list
        print("filter_list del nul:")
        print(args.filter_list)
    if args.accumulate == False:
        task_info_j["accumulate"] = 0
    if task_info_j != {}:
        task_info_s = json.dumps(task_info_j)
        bytesString = task_info_s.encode(encoding="utf-8")
        task_info = base64.b64encode(bytesString).decode('UTF-8')
    else:
        task_info = None

    # 修改入口文件
    if args.entrance_type == 'main':
        modify_main(args.main_path, task_info)
    elif args.entrance_type == 'spp':
        modify_spp_handle_init(args.main_path, task_info)

    work_space = get_workspace()
    get_makefile_list(work_space)

    if args.makefile_list is not None:
        for i in args.makefile_list:
            make_file_list.append(i)

    print("make file list")
    print(make_file_list)

    for i in make_file_list:
        makefile_path = os.path.join(work_space, i)
        print("makefile_path:", makefile_path)
        modify_makefile(makefile_path)

    get_cmake_list(work_space)
    print("cmake list")
    print(cmake_list)
    for i in cmake_list:
        modify_cmakelists(i)

