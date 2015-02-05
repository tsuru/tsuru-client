import re
import os
import json
from subprocess import check_output

from docutils import nodes
from docutils.parsers import rst
from docutils.parsers.rst.directives import unchanged


class CommandNode(nodes.Element):
    pass


class CommandDirective(rst.Directive):
    has_content = False
    final_argument_whitespace = True
    required_arguments = 1
    option_spec = dict(title=unchanged)

    def run(self):
        env = self.state.document.settings.env
        node = CommandNode()
        node.line = self.lineno
        node['command'] = self.arguments[0]
        node['title'] = self.options.get('title', '')
        node['use_shell'] = True
        node['hide_standard_error'] = False
        _, cwd = env.relfn2path(self.options.get('cwd', '/'))
        node['working_directory'] = cwd
        return [node]


def run_programs(app, doctree):

    for node in doctree.traverse(CommandNode):
        command = node['command']
        try:
            command_data = app.env.tsuru_json[command]
        except KeyError:
            error_message = 'Command {0} not found'.format(command)
            error_node = doctree.reporter.error(error_message, base_node=node)
            node.replace_self(error_node)
            continue
        output = "$ {}".format(command_data['usage'])

        all_nodes = []

        title = node.get('title')
        if title:
            all_nodes.append(nodes.subtitle(title, title))

        new_node = nodes.literal_block(output, output)
        new_node['language'] = 'text'
        all_nodes.append(new_node)

        remaining_output = command_data['desc']
        remaining_output = remaining_output.replace("\n", "<br>")
        paragraph = nodes.paragraph()
        raw = nodes.raw('', remaining_output, format='html')
        paragraph.append(raw)

        all_nodes.append(paragraph)

        node.replace_self(all_nodes)


def read_cmds(app):
    if not hasattr(app.env, 'tsuru_json'):
        destination = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'cmds.json')
        with open(destination, 'r') as f:
            data = json.loads(f.read())
        app.env.tsuru_json = data


def setup(app):
    app.add_directive('tsuru-command', CommandDirective)
    app.connect(str('builder-inited'), read_cmds)
    app.connect(str('doctree-read'), run_programs)


def main():
    parts_regex = re.compile(r'tsuru version.*Usage: (.*?)\n+(.*)', re.DOTALL)

    result = check_output("tsuru | egrep \"^[  ]\" | awk -F ' ' '{print $1}'", shell=True)
    cmds = result.split('\n')
    final_result = {}
    for cmd in cmds:
        if not cmd:
            continue
        result = check_output('tsuru help {}'.format(cmd), shell=True)
        matchdata = parts_regex.match(result)
        if matchdata is None:
            print "Ignored command: {}".format(cmd)
            continue
        result = {
            'usage': matchdata.group(1),
            'desc': matchdata.group(2),
        }
        final_result[cmd] = result
    out = json.dumps(final_result, indent=2)
    destination = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'cmds.json')
    with open(destination, 'wb') as f:
        f.write(out)


if __name__ == '__main__':
    main()
