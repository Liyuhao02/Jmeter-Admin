/**
 * JMX Parser - JMeter JMX 文件解析和序列化工具
 * 
 * 提供 JMeter JMX XML 文件的解析、序列化和元数据管理功能
 */

// ============================================
// 元素元数据定义
// ============================================

const ELEMENT_META = {
  // ==================== 测试计划 ====================
  TestPlan: {
    label: '测试计划',
    icon: 'Document',
    isLeaf: false,
    properties: [
      { key: 'TestPlan.serialize_threadgroups', label: '序列化线程组', type: 'boolean' },
      { key: 'TestPlan.functional_mode', label: '功能模式', type: 'boolean' },
    ]
  },

  // ==================== 线程组 ====================
  ThreadGroup: {
    label: '线程组',
    icon: 'User',
    isLeaf: false,
    summaryKeys: [
      { key: 'ThreadGroup.num_threads', label: '线程' },
      { key: 'LoopController.loops', label: '循环' }
    ],
    properties: [
      { key: 'ThreadGroup.num_threads', label: '线程数', type: 'number' },
      { key: 'ThreadGroup.ramp_time', label: 'Ramp-Up时间(秒)', type: 'number' },
      { key: 'ThreadGroup.on_sample_error', label: '错误处理', type: 'select', options: [
        { value: 'continue', label: '继续' },
        { value: 'startnextloop', label: '开始下一循环' },
        { value: 'stopthread', label: '停止线程' },
        { value: 'stoptest', label: '停止测试' },
        { value: 'stoptestnow', label: '立即停止测试' }
      ]},
      { key: 'LoopController.loops', label: '循环次数', type: 'number', min: -1, nested: 'ThreadGroup.main_controller' },
      { key: 'LoopController.continue_forever', label: '永远循环', type: 'boolean', nested: 'ThreadGroup.main_controller' },
    ]
  },
  SetupThreadGroup: {
    label: 'setUp线程组',
    icon: 'User',
    isLeaf: false,
    summaryKeys: [
      { key: 'ThreadGroup.num_threads', label: '线程' },
      { key: 'LoopController.loops', label: '循环' }
    ],
    properties: [
      { key: 'ThreadGroup.num_threads', label: '线程数', type: 'number' },
      { key: 'ThreadGroup.ramp_time', label: 'Ramp-Up时间(秒)', type: 'number' },
      { key: 'ThreadGroup.on_sample_error', label: '错误处理', type: 'select', options: [
        { value: 'continue', label: '继续' },
        { value: 'startnextloop', label: '开始下一循环' },
        { value: 'stopthread', label: '停止线程' },
        { value: 'stoptest', label: '停止测试' },
        { value: 'stoptestnow', label: '立即停止测试' }
      ]},
      { key: 'LoopController.loops', label: '循环次数', type: 'number', min: -1, nested: 'ThreadGroup.main_controller' },
    ]
  },
  PostThreadGroup: {
    label: 'tearDown线程组',
    icon: 'User',
    isLeaf: false,
    summaryKeys: [
      { key: 'ThreadGroup.num_threads', label: '线程' },
      { key: 'LoopController.loops', label: '循环' }
    ],
    properties: [
      { key: 'ThreadGroup.num_threads', label: '线程数', type: 'number' },
      { key: 'ThreadGroup.ramp_time', label: 'Ramp-Up时间(秒)', type: 'number' },
      { key: 'ThreadGroup.on_sample_error', label: '错误处理', type: 'select', options: [
        { value: 'continue', label: '继续' },
        { value: 'startnextloop', label: '开始下一循环' },
        { value: 'stopthread', label: '停止线程' },
        { value: 'stoptest', label: '停止测试' },
        { value: 'stoptestnow', label: '立即停止测试' }
      ]},
      { key: 'LoopController.loops', label: '循环次数', type: 'number', min: -1, nested: 'ThreadGroup.main_controller' },
    ]
  },
  UltimateThreadGroup: {
    label: '终极线程组 (Ultimate Thread Group)',
    icon: 'User',
    isLeaf: false,
    summaryKeys: [], // 特殊处理，无简单摘要
    properties: [
      { key: 'ThreadGroup.on_sample_error', label: '错误处理', type: 'select', options: [
        { value: 'continue', label: '继续' },
        { value: 'startnextloop', label: '开始下一循环' },
        { value: 'stopthread', label: '停止线程' },
        { value: 'stoptest', label: '停止测试' },
        { value: 'stoptestnow', label: '立即停止测试' }
      ]},
      { key: 'ultimatethreadgroupdata', label: '线程调度配置', type: 'threadSchedule' }
    ]
  },

  // ==================== 采样器 ====================
  HTTPSamplerProxy: {
    label: 'HTTP请求',
    icon: 'Link',
    isLeaf: false,
    summaryKeys: [
      { key: 'HTTPSampler.method', label: '' },
      { key: 'HTTPSampler.domain', label: '' },
      { key: 'HTTPSampler.path', label: '' }
    ],
    summary: (props) => `${props['HTTPSampler.method'] || 'GET'} ${props['HTTPSampler.path'] || '/'}`,
    properties: [
      { key: 'HTTPSampler.protocol', label: '协议', type: 'select', options: [
        { value: 'http', label: 'HTTP' }, { value: 'https', label: 'HTTPS' }
      ]},
      { key: 'HTTPSampler.domain', label: '服务器名称或IP', type: 'string' },
      { key: 'HTTPSampler.port', label: '端口号', type: 'number' },
      { key: 'HTTPSampler.path', label: '路径', type: 'string' },
      { key: 'HTTPSampler.method', label: '请求方法', type: 'select', options: [
        { value: 'GET', label: 'GET' }, { value: 'POST', label: 'POST' },
        { value: 'PUT', label: 'PUT' }, { value: 'DELETE', label: 'DELETE' },
        { value: 'PATCH', label: 'PATCH' }, { value: 'HEAD', label: 'HEAD' },
        { value: 'OPTIONS', label: 'OPTIONS' }
      ]},
      { key: 'HTTPSampler.contentEncoding', label: '内容编码', type: 'string' },
      { key: 'HTTPSampler.postBodyRaw', label: '使用原始Body', type: 'boolean' },
      { key: 'HTTPSampler.body', label: '请求Body', type: 'textarea', special: 'httpBody' },
    ]
  },
  DebugSampler: {
    label: '调试采样器',
    icon: 'Monitor',
    isLeaf: false,
    properties: [
      { key: 'displayJMeterProperties', label: '显示JMeter属性', type: 'boolean' },
      { key: 'displayJMeterVariables', label: '显示JMeter变量', type: 'boolean' },
      { key: 'displaySystemProperties', label: '显示系统属性', type: 'boolean' },
    ]
  },
  JDBCSampler: {
    label: 'JDBC请求',
    icon: 'Coin',
    isLeaf: false,
    summaryKeys: [{ key: 'query', label: 'SQL' }],
    properties: [
      { key: 'dataSource', label: '数据源名称', type: 'string' },
      { key: 'queryType', label: '查询类型', type: 'select', options: [
        { value: 'Select Statement', label: 'Select' },
        { value: 'Update Statement', label: 'Update' },
        { value: 'Callable Statement', label: 'Callable' },
        { value: 'Prepared Select Statement', label: 'Prepared Select' },
        { value: 'Prepared Update Statement', label: 'Prepared Update' },
      ]},
      { key: 'query', label: 'SQL查询', type: 'textarea' },
      { key: 'queryArguments', label: '参数值', type: 'string' },
      { key: 'queryArgumentsTypes', label: '参数类型', type: 'string' },
      { key: 'resultVariable', label: '结果变量名', type: 'string' },
      { key: 'queryTimeout', label: '查询超时(秒)', type: 'number' },
    ]
  },
  JSR223Sampler: {
    label: 'JSR223采样器',
    icon: 'EditPen',
    isLeaf: false,
    category: 'sampler',
    properties: [
      { key: 'scriptLanguage', label: '脚本语言', type: 'select', options: [
        { value: 'groovy', label: 'Groovy' },
        { value: 'javascript', label: 'JavaScript' },
        { value: 'beanshell', label: 'BeanShell' },
      ]},
      { key: 'filename', label: '脚本文件', type: 'string' },
      { key: 'parameters', label: '参数', type: 'string' },
      { key: 'script', label: '脚本', type: 'textarea' },
      { key: 'cacheKey', label: '缓存Key', type: 'string' },
    ]
  },
  TCPSampler: {
    label: 'TCP采样器',
    icon: 'Monitor',
    isLeaf: false,
    category: 'sampler',
    properties: [
      { key: 'TCPSampler.server', label: '服务器名称或IP', type: 'string' },
      { key: 'TCPSampler.port', label: '端口号', type: 'number' },
      { key: 'TCPSampler.ctimeout', label: '连接超时(ms)', type: 'number', defaultValue: '' },
      { key: 'TCPSampler.timeout', label: '响应超时(ms)', type: 'number', defaultValue: '' },
      { key: 'TCPSampler.reUseConnection', label: '重用连接', type: 'boolean', defaultValue: 'true' },
      { key: 'TCPSampler.nodelay', label: '设置NoDelay', type: 'boolean', defaultValue: 'false' },
      { key: 'TCPSampler.EolByte', label: '行尾(EOL)字节值', type: 'number', defaultValue: '0' },
      { key: 'TCPSampler.request', label: '要发送的文本', type: 'textarea' },
      { key: 'TCPSampler.closeConnection', label: '关闭连接', type: 'boolean', defaultValue: 'false' }
    ],
    summary: (props) => `${props['TCPSampler.server'] || ''}:${props['TCPSampler.port'] || ''}`
  },

  // ==================== 配置元素 ====================
  CSVDataSet: {
    label: 'CSV数据文件',
    icon: 'Grid',
    isLeaf: true,
    summaryKeys: [
      { key: 'filename', label: '文件' },
      { key: 'variableNames', label: '变量' }
    ],
    properties: [
      { key: 'filename', label: '文件名', type: 'string' },
      { key: 'variableNames', label: '变量名(逗号分隔)', type: 'string' },
      { key: 'delimiter', label: '分隔符', type: 'string' },
      { key: 'fileEncoding', label: '文件编码', type: 'string' },
      { key: 'ignoreFirstLine', label: '忽略首行', type: 'boolean' },
      { key: 'recycle', label: '是否循环', type: 'boolean' },
      { key: 'stopThread', label: '遇到EOF停止线程', type: 'boolean' },
      { key: 'shareMode', label: '共享模式', type: 'select', options: [
        { value: 'shareMode.all', label: '所有线程' },
        { value: 'shareMode.group', label: '当前线程组' },
        { value: 'shareMode.thread', label: '当前线程' }
      ]},
    ]
  },
  HeaderManager: {
    label: 'HTTP头管理器',
    icon: 'List',
    isLeaf: true,
    summaryKeys: [], // 键值对，无简单摘要
    properties: [
      { key: 'HeaderManager.headers', label: 'HTTP头列表', type: 'keyValueList', itemKeys: ['Header.name', 'Header.value'], itemLabels: ['名称', '值'] }
    ]
  },
  Arguments: {
    label: '用户定义变量',
    icon: 'Setting',
    isLeaf: true,
    properties: [
      { key: 'Arguments.arguments', label: '变量列表', type: 'keyValueList', itemKeys: ['Argument.name', 'Argument.value'], itemLabels: ['变量名', '变量值'] }
    ]
  },
  CookieManager: {
    label: 'Cookie管理器',
    icon: 'Tickets',
    isLeaf: true,
    properties: [
      { key: 'CookieManager.policy', label: '清除策略', type: 'select', options: [
        { value: 'default', label: '标准' }, { value: 'compatibility', label: '兼容' },
        { value: 'netscape', label: 'Netscape' }, { value: 'ignoreCookies', label: '忽略Cookie' }
      ]},
      { key: 'CookieManager.clearEachIteration', label: '每次迭代清除', type: 'boolean' },
    ]
  },
  ConfigTestElement: {
    label: 'HTTP默认值',
    icon: 'Setting',
    isLeaf: true,
    properties: [
      { key: 'HTTPSampler.domain', label: '服务器名称或IP', type: 'string' },
      { key: 'HTTPSampler.port', label: '端口号', type: 'number' },
      { key: 'HTTPSampler.protocol', label: '协议', type: 'select', options: [
        { value: 'http', label: 'HTTP' }, { value: 'https', label: 'HTTPS' }
      ]},
      { key: 'HTTPSampler.path', label: '路径', type: 'string' },
      { key: 'HTTPSampler.contentEncoding', label: '编码', type: 'string' },
      { key: 'HTTPSampler.connect_timeout', label: '连接超时(ms)', type: 'number' },
      { key: 'HTTPSampler.response_timeout', label: '响应超时(ms)', type: 'number' },
    ]
  },
  DataSourceElement: {
    label: 'JDBC连接配置',
    icon: 'Connection',
    isLeaf: true,
    category: 'config',
    properties: [
      { key: 'dataSource', label: '数据源名称', type: 'string' },
      { key: 'dbUrl', label: '数据库URL', type: 'string' },
      { key: 'driver', label: '驱动类', type: 'string' },
      { key: 'username', label: '用户名', type: 'string' },
      { key: 'password', label: '密码', type: 'string' },
      { key: 'maxActive', label: '最大连接数', type: 'number' },
      { key: 'maxIdle', label: '最大空闲数', type: 'number' },
      { key: 'maxWait', label: '最大等待(ms)', type: 'number' },
    ]
  },
  AuthManager: {
    label: 'HTTP授权管理器',
    icon: 'Lock',
    isLeaf: true,
    category: 'config',
    properties: [
      {
        key: 'AuthManager.auth_list',
        label: '授权列表',
        type: 'keyValueList',
        itemKeys: ['Authorization.url', 'Authorization.username', 'Authorization.password', 'Authorization.domain', 'Authorization.realm'],
        itemLabels: ['基础URL', '用户名', '密码', '域', '域(Realm)']
      },
      { key: 'AuthManager.controlledByThreadGroup', label: '由线程组控制', type: 'boolean', defaultValue: 'false' },
      { key: 'AuthManager.clearEachIteration', label: '每次迭代清除', type: 'boolean', defaultValue: 'false' }
    ]
  },
  CacheManager: {
    label: 'HTTP缓存管理器',
    icon: 'FolderOpened',
    isLeaf: true,
    category: 'config',
    properties: [
      { key: 'clearEachIteration', label: '每次迭代清除缓存', type: 'boolean', defaultValue: 'false' },
      { key: 'useExpires', label: '使用Cache-Control/Expires', type: 'boolean', defaultValue: 'true' },
      { key: 'maxSize', label: '缓存最大数量', type: 'number', defaultValue: '5000' },
      { key: 'controlledByThread', label: '由线程组控制', type: 'boolean', defaultValue: 'false' }
    ]
  },
  DNSCacheManager: {
    label: 'DNS缓存管理器',
    icon: 'Cloudy',
    isLeaf: true,
    category: 'config',
    properties: [
      { key: 'DNSCacheManager.clearEachIteration', label: '每次迭代清除', type: 'boolean', defaultValue: 'false' },
      { key: 'DNSCacheManager.isCustomResolver', label: '使用自定义DNS解析器', type: 'boolean', defaultValue: 'false' }
    ]
  },

  // ==================== 定时器 ====================
  ConstantTimer: {
    label: '固定定时器',
    icon: 'Timer',
    isLeaf: true,
    summaryKeys: [{ key: 'ConstantTimer.delay', label: '延迟(ms)' }],
    properties: [
      { key: 'ConstantTimer.delay', label: '延迟时间(毫秒)', type: 'number' },
    ]
  },
  UniformRandomTimer: {
    label: '均匀随机定时器',
    icon: 'Timer',
    isLeaf: true,
    summaryKeys: [
      { key: 'ConstantTimer.delay', label: '基准(ms)' },
      { key: 'RandomTimer.range', label: '随机(ms)' }
    ],
    properties: [
      { key: 'ConstantTimer.delay', label: '固定延迟(ms)', type: 'number' },
      { key: 'RandomTimer.range', label: '随机延迟最大值(ms)', type: 'number' },
    ]
  },
  GaussianRandomTimer: {
    label: '高斯随机定时器',
    icon: 'Timer',
    isLeaf: true,
    category: 'timer',
    properties: [
      { key: 'ConstantTimer.delay', label: '固定延迟(ms)', type: 'number' },
      { key: 'RandomTimer.range', label: '偏差(ms)', type: 'number' },
    ]
  },
  ConstantThroughputTimer: {
    label: '常量吞吐量定时器',
    icon: 'Odometer',
    isLeaf: true,
    summaryKeys: [{ key: 'throughput', label: '吞吐量/min' }],
    category: 'timer',
    properties: [
      { key: 'throughput', label: '目标吞吐量(samples/min)', type: 'number', defaultValue: '60.0' },
      { key: 'calcMode', label: '基于计算', type: 'select', defaultValue: '0',
        options: [
          { label: '仅此线程', value: '0' },
          { label: '所有活动线程', value: '1' },
          { label: '当前线程组所有活动线程', value: '2' },
          { label: '所有活动线程(共享)', value: '3' },
          { label: '当前线程组所有活动线程(共享)', value: '4' }
        ]
      }
    ]
  },
  SynchronizingTimer: {
    label: '集合点(同步定时器)',
    icon: 'Connection',
    isLeaf: true,
    category: 'timer',
    properties: [
      { key: 'groupSize', label: '模拟用户组大小(0=全部)', type: 'number', defaultValue: '0' },
      { key: 'timeoutInMs', label: '超时时间(ms,0=无限)', type: 'number', defaultValue: '0' }
    ]
  },
  PoissonRandomTimer: {
    label: '泊松随机定时器',
    icon: 'Timer',
    isLeaf: true,
    category: 'timer',
    properties: [
      { key: 'ConstantTimer.delay', label: 'Lambda值(ms)', type: 'number', defaultValue: '300' },
      { key: 'RandomTimer.range', label: '偏移量(ms)', type: 'number', defaultValue: '100' }
    ]
  },

  // ==================== 断言 ====================
  ResponseAssertion: {
    label: '响应断言',
    icon: 'Check',
    isLeaf: true,
    summaryKeys: [{ key: 'Asserion.test_strings', label: '匹配' }],
    properties: [
      { key: 'Assertion.test_field', label: '测试字段', type: 'select', options: [
        { value: 'Assertion.response_data', label: '响应文本' },
        { value: 'Assertion.response_code', label: '响应代码' },
        { value: 'Assertion.response_message', label: '响应信息' },
        { value: 'Assertion.response_headers', label: '响应头' },
        { value: 'Assertion.request_headers', label: '请求头' },
        { value: 'Assertion.request_data', label: '请求数据' },
        { value: 'Assertion.sample_label', label: 'URL样本' },
      ]},
      { key: 'Assertion.test_type', label: '匹配规则', type: 'select', options: [
        { value: '2', label: '包括 (Contains)' },
        { value: '1', label: '匹配 (Matches)' },
        { value: '8', label: '字符串 (Substring)' },
        { value: '4', label: '相等 (Equals)' },
        { value: '18', label: '不包括 (NOT Contains)' },
        { value: '17', label: '不匹配 (NOT Matches)' },
        { value: '24', label: '不是字符串 (NOT Substring)' },
        { value: '20', label: '不相等 (NOT Equals)' },
      ]},
      // 注意：JMeter 源码中的拼写错误是 "Asserion" 而非 "Assertion"
      { key: 'Asserion.test_strings', label: '断言字符串', type: 'stringList', description: '断言匹配字符串列表（每行一个）' },
    ]
  },
  JSONPathAssertion: {
    label: 'JSON断言',
    icon: 'Check',
    isLeaf: true,
    summaryKeys: [{ key: 'JSON_PATH', label: 'JSONPath' }],
    properties: [
      { key: 'JSON_PATH', label: 'JSON路径', type: 'string' },
      { key: 'EXPECTED_VALUE', label: '期望值', type: 'string' },
      { key: 'JSONVALIDATION', label: '验证值', type: 'boolean' },
      { key: 'EXPECT_NULL', label: '期望为null', type: 'boolean' },
      { key: 'INVERT', label: '反转结果', type: 'boolean' },
    ]
  },
  DurationAssertion: {
    label: '持续时间断言',
    icon: 'Timer',
    isLeaf: true,
    properties: [
      { key: 'DurationAssertion.duration', label: '允许的最大持续时间(ms)', type: 'number' },
    ]
  },
  SizeAssertion: {
    label: '大小断言',
    icon: 'Histogram',
    isLeaf: true,
    category: 'assertion',
    properties: [
      { key: 'SizeAssertion.size', label: '字节大小', type: 'number' },
      { key: 'SizeAssertion.operator', label: '比较类型', type: 'select', options: [
        { value: '1', label: '等于' },
        { value: '2', label: '大于' },
        { value: '3', label: '小于' },
        { value: '4', label: '不等于' },
        { value: '5', label: '大于等于' },
        { value: '6', label: '小于等于' },
      ]},
    ]
  },
  JSR223Assertion: {
    label: 'JSR223断言',
    icon: 'Check',
    isLeaf: true,
    category: 'assertion',
    properties: [
      { key: 'scriptLanguage', label: '脚本语言', type: 'select', defaultValue: 'groovy',
        options: [
          { label: 'groovy', value: 'groovy' },
          { label: 'javascript', value: 'javascript' },
          { label: 'jexl3', value: 'jexl3' },
          { label: 'beanshell', value: 'beanshell' }
        ]
      },
      { key: 'script', label: '脚本', type: 'textarea' },
      { key: 'parameters', label: '参数', type: 'string' },
      { key: 'filename', label: '脚本文件', type: 'string' },
      { key: 'cacheKey', label: '缓存编译脚本Key', type: 'string', defaultValue: 'true' }
    ]
  },

  // ==================== 提取器 ====================
  RegexExtractor: {
    label: '正则提取器',
    icon: 'Search',
    isLeaf: true,
    summaryKeys: [
      { key: 'RegexExtractor.refname', label: '变量' },
      { key: 'RegexExtractor.regex', label: '正则' }
    ],
    properties: [
      { key: 'RegexExtractor.useHeaders', label: '应用到', type: 'select', options: [
        { value: 'false', label: '响应Body' },
        { value: 'true', label: '响应头' },
        { value: 'URL', label: 'URL' },
        { value: 'code', label: '响应代码' },
        { value: 'message', label: '响应信息' },
      ]},
      { key: 'RegexExtractor.refname', label: '引用名称', type: 'string' },
      { key: 'RegexExtractor.regex', label: '正则表达式', type: 'string' },
      { key: 'RegexExtractor.template', label: '模板', type: 'string' },
      { key: 'RegexExtractor.match_number', label: '匹配数字', type: 'string' },
      { key: 'RegexExtractor.default', label: '默认值', type: 'string' },
    ]
  },
  JSONPostProcessor: {
    label: 'JSON提取器',
    icon: 'Search',
    isLeaf: true,
    summaryKeys: [
      { key: 'JSONPostProcessor.referenceNames', label: '变量' },
      { key: 'JSONPostProcessor.jsonPathExprs', label: 'JSONPath' }
    ],
    properties: [
      { key: 'JSONPostProcessor.referenceNames', label: '变量名', type: 'string' },
      { key: 'JSONPostProcessor.jsonPathExprs', label: 'JSON路径表达式', type: 'string' },
      { key: 'JSONPostProcessor.match_numbers', label: '匹配数字', type: 'string' },
      { key: 'JSONPostProcessor.defaultValues', label: '默认值', type: 'string' },
    ]
  },
  XPathExtractor: {
    label: 'XPath提取器',
    icon: 'Search',
    isLeaf: true,
    properties: [
      { key: 'XPathExtractor.refname', label: '引用名称', type: 'string' },
      { key: 'XPathExtractor.xpathQuery', label: 'XPath表达式', type: 'string' },
      { key: 'XPathExtractor.default', label: '默认值', type: 'string' },
      { key: 'XPathExtractor.matchNumber', label: '匹配数字', type: 'number' },
    ]
  },
  BoundaryExtractor: {
    label: '边界提取器',
    icon: 'Search',
    isLeaf: true,
    properties: [
      { key: 'BoundaryExtractor.refname', label: '引用名称', type: 'string' },
      { key: 'BoundaryExtractor.lboundary', label: '左边界', type: 'string' },
      { key: 'BoundaryExtractor.rboundary', label: '右边界', type: 'string' },
      { key: 'BoundaryExtractor.match_number', label: '匹配数字', type: 'string' },
      { key: 'BoundaryExtractor.default', label: '默认值', type: 'string' },
    ]
  },

  // ==================== 前置/后置处理器 ====================
  BeanShellPreProcessor: {
    label: 'BeanShell前置处理器',
    icon: 'EditPen',
    isLeaf: true,
    properties: [
      { key: 'filename', label: '脚本文件', type: 'string' },
      { key: 'parameters', label: '参数', type: 'string' },
      { key: 'script', label: '脚本', type: 'textarea' },
    ]
  },
  BeanShellPostProcessor: {
    label: 'BeanShell后置处理器',
    icon: 'EditPen',
    isLeaf: true,
    properties: [
      { key: 'filename', label: '脚本文件', type: 'string' },
      { key: 'parameters', label: '参数', type: 'string' },
      { key: 'script', label: '脚本', type: 'textarea' },
    ]
  },
  JSR223PreProcessor: {
    label: 'JSR223前置处理器',
    icon: 'EditPen',
    isLeaf: true,
    properties: [
      { key: 'scriptLanguage', label: '脚本语言', type: 'select', options: [
        { value: 'groovy', label: 'Groovy' },
        { value: 'javascript', label: 'JavaScript' },
        { value: 'beanshell', label: 'BeanShell' },
      ]},
      { key: 'filename', label: '脚本文件', type: 'string' },
      { key: 'parameters', label: '参数', type: 'string' },
      { key: 'script', label: '脚本', type: 'textarea' },
      { key: 'cacheKey', label: '缓存Key', type: 'string' },
    ]
  },
  JSR223PostProcessor: {
    label: 'JSR223后置处理器',
    icon: 'EditPen',
    isLeaf: true,
    properties: [
      { key: 'scriptLanguage', label: '脚本语言', type: 'select', options: [
        { value: 'groovy', label: 'Groovy' },
        { value: 'javascript', label: 'JavaScript' },
        { value: 'beanshell', label: 'BeanShell' },
      ]},
      { key: 'filename', label: '脚本文件', type: 'string' },
      { key: 'parameters', label: '参数', type: 'string' },
      { key: 'script', label: '脚本', type: 'textarea' },
      { key: 'cacheKey', label: '缓存Key', type: 'string' },
    ]
  },

  // ==================== 监听器 ====================
  ResultCollector: {
    label: '结果收集器',
    icon: 'DataAnalysis',
    isLeaf: true,
    properties: [
      { key: 'filename', label: '输出文件名', type: 'string' },
      { key: 'ResultCollector.error_logging', label: '仅记录错误', type: 'boolean' },
      { key: 'ResultCollector.success_only_logging', label: '仅记录成功', type: 'boolean' },
    ]
  },
  JSR223Listener: {
    label: 'JSR223 监听器',
    icon: 'DataLine',
    isLeaf: true,
    category: 'listener',
    properties: [
      { key: 'scriptLanguage', label: '脚本语言', type: 'select', options: [
        { value: 'groovy', label: 'Groovy' },
        { value: 'javascript', label: 'JavaScript' },
        { value: 'beanshell', label: 'BeanShell' },
        { value: 'jexl3', label: 'JEXL3' }
      ]},
      { key: 'script', label: '脚本内容', type: 'textarea' },
      { key: 'parameters', label: '参数', type: 'string' },
      { key: 'filename', label: '脚本文件路径', type: 'string' },
      { key: 'cacheKey', label: '缓存编译脚本', type: 'select', options: [
        { value: 'true', label: '是' },
        { value: 'false', label: '否' }
      ]}
    ]
  },
  BackendListener: {
    label: '后端监听器',
    icon: 'Cpu',
    isLeaf: true,
    category: 'listener',
    properties: [
      { key: 'classname', label: '实现类', type: 'select', defaultValue: 'org.apache.jmeter.visualizers.backend.influxdb.InfluxdbBackendListenerClient',
        options: [
          { label: 'InfluxDB', value: 'org.apache.jmeter.visualizers.backend.influxdb.InfluxdbBackendListenerClient' },
          { label: 'Graphite', value: 'org.apache.jmeter.visualizers.backend.graphite.GraphiteBackendListenerClient' }
        ]
      },
      { key: 'QUEUE_SIZE', label: '队列大小', type: 'number', defaultValue: '5000' }
    ]
  },
  Summariser: {
    label: '汇总报告',
    icon: 'Document',
    isLeaf: true,
    category: 'listener',
    properties: [
      { key: 'Summariser.name', label: '名称', type: 'string', defaultValue: 'summary' }
    ]
  },

  // ==================== 控制器 ====================
  LoopController: {
    label: '循环控制器',
    icon: 'RefreshRight',
    isLeaf: false,
    summaryKeys: [{ key: 'LoopController.loops', label: '次数' }],
    properties: [
      { key: 'LoopController.loops', label: '循环次数', type: 'number', min: -1 },
      { key: 'LoopController.continue_forever', label: '永远循环', type: 'boolean' },
    ]
  },
  IfController: {
    label: '条件控制器',
    icon: 'Switch',
    isLeaf: false,
    summaryKeys: [{ key: 'IfController.condition', label: '条件' }],
    properties: [
      { key: 'IfController.condition', label: '条件表达式', type: 'string' },
      { key: 'IfController.evaluateAll', label: '评估所有子元素', type: 'boolean' },
    ]
  },
  WhileController: {
    label: 'While控制器',
    icon: 'RefreshRight',
    isLeaf: false,
    properties: [
      { key: 'WhileController.condition', label: '条件', type: 'string' },
    ]
  },
  ForeachController: {
    label: 'ForEach控制器',
    icon: 'RefreshRight',
    isLeaf: false,
    properties: [
      { key: 'ForeachController.inputVal', label: '输入变量前缀', type: 'string' },
      { key: 'ForeachController.returnVal', label: '输出变量名', type: 'string' },
      { key: 'ForeachController.useSeparator', label: '使用分隔符', type: 'boolean' },
      { key: 'ForeachController.startIndex', label: '起始索引', type: 'number' },
      { key: 'ForeachController.endIndex', label: '结束索引', type: 'number' },
    ]
  },
  OnceOnlyController: {
    label: '仅一次控制器',
    icon: 'Stamp',
    isLeaf: false,
    properties: []
  },
  RandomController: {
    label: '随机控制器',
    icon: 'MagicStick',
    isLeaf: false,
    properties: [
      { key: 'InterleaveControl.style', label: '忽略子控制器块', type: 'boolean' },
    ]
  },
  TransactionController: {
    label: '事务控制器',
    icon: 'Folder',
    isLeaf: false,
    summaryKeys: [{ key: 'TransactionController.includeTimers', label: '含Timer' }],
    properties: [
      { key: 'TransactionController.includeTimers', label: '包含定时器', type: 'boolean' },
      { key: 'TransactionController.parent', label: '生成父样本', type: 'boolean' },
    ]
  },
  CriticalSectionController: {
    label: '临界部分控制器',
    icon: 'Lock',
    isLeaf: false,
    category: 'controller',
    properties: [
      { key: 'CriticalSectionController.lockName', label: '锁名称', type: 'string' }
    ]
  },
  ThroughputController: {
    label: '吞吐量控制器',
    icon: 'DataAnalysis',
    isLeaf: false,
    category: 'controller',
    properties: [
      { key: 'style', label: '执行模式', type: 'select', defaultValue: '0',
        options: [
          { label: '百分比执行', value: '0' },
          { label: '总执行次数', value: '1' }
        ]
      },
      { key: 'maxThroughput', label: '吞吐量', type: 'number', defaultValue: '100' },
      { key: 'perThread', label: '每用户', type: 'boolean', defaultValue: 'false' }
    ]
  },
  SwitchController: {
    label: '开关控制器',
    icon: 'Switch',
    isLeaf: false,
    category: 'controller',
    properties: [
      { key: 'SwitchController.value', label: '切换值', type: 'string', defaultValue: '' }
    ],
    summary: (props) => props['SwitchController.value'] || '默认'
  },
  InterleaveController: {
    label: '交替控制器',
    icon: 'Sort',
    isLeaf: false,
    category: 'controller',
    properties: [
      { key: 'InterleaveController.style', label: '忽略子控制器块', type: 'boolean', defaultValue: 'false' },
      { key: 'InterleaveController.accrossThreads', label: '线程间交替', type: 'boolean', defaultValue: 'false' }
    ]
  },
  RandomOrderController: {
    label: '随机顺序控制器',
    icon: 'Opportunity',
    isLeaf: false,
    category: 'controller',
    properties: []
  },
  RunTime: {
    label: '运行时间控制器',
    icon: 'Clock',
    isLeaf: false,
    category: 'controller',
    properties: [
      { key: 'RunTime.seconds', label: '运行时间(秒)', type: 'number', defaultValue: '1' }
    ],
    summary: (props) => `${props['RunTime.seconds'] || '1'}s`
  },
}

// ============================================
// 元数据访问函数
// ============================================

/**
 * 获取元素的元数据定义
 * @param {string} testclass - JMeter 元素类型
 * @returns {object|null} 元数据定义对象
 */
export function getElementMeta(testclass) {
  // 处理带包名的 testclass（如 kg.apc.jmeter.threads.UltimateThreadGroup）
  const shortName = testclass.includes('.') ? testclass.split('.').pop() : testclass
  return ELEMENT_META[shortName] || ELEMENT_META[testclass] || null
}

/**
 * 推断元素的类别
 * @param {string} testclass - JMeter 元素类型
 * @returns {string} 类别名称
 */
export function getElementCategory(testclass) {
  const meta = getElementMeta(testclass)
  if (meta && meta.category) {
    return meta.category
  }
  
  // 根据 testclass 名称推断类别
  const shortName = testclass.includes('.') ? testclass.split('.').pop() : testclass
  
  // 线程组
  const threadGroups = ['ThreadGroup', 'SetupThreadGroup', 'PostThreadGroup', 'UltimateThreadGroup']
  if (threadGroups.includes(shortName)) return 'threadGroup'
  
  // 采样器
  const samplers = ['HTTPSamplerProxy', 'DebugSampler', 'JDBCSampler', 'JSR223Sampler', 'TCPSampler']
  if (samplers.includes(shortName)) return 'sampler'
  
  // 配置元素
  const configs = ['CSVDataSet', 'HeaderManager', 'Arguments', 'CookieManager', 'ConfigTestElement', 
                   'DataSourceElement', 'AuthManager', 'CacheManager', 'DNSCacheManager']
  if (configs.includes(shortName)) return 'config'
  
  // 定时器
  const timers = ['ConstantTimer', 'UniformRandomTimer', 'GaussianRandomTimer', 
                  'ConstantThroughputTimer', 'SynchronizingTimer', 'PoissonRandomTimer']
  if (timers.includes(shortName)) return 'timer'
  
  // 断言
  const assertions = ['ResponseAssertion', 'JSONPathAssertion', 'DurationAssertion', 
                      'SizeAssertion', 'JSR223Assertion']
  if (assertions.includes(shortName)) return 'assertion'
  
  // 提取器（后置处理器）
  const extractors = ['RegexExtractor', 'JSONPostProcessor', 'XPathExtractor', 'BoundaryExtractor']
  if (extractors.includes(shortName)) return 'postProcessor'
  
  // 前置处理器
  const preProcessors = ['BeanShellPreProcessor', 'JSR223PreProcessor']
  if (preProcessors.includes(shortName)) return 'preProcessor'
  
  // 后置处理器
  const postProcessors = ['BeanShellPostProcessor', 'JSR223PostProcessor']
  if (postProcessors.includes(shortName)) return 'postProcessor'
  
  // 监听器
  const listeners = ['ResultCollector', 'JSR223Listener', 'BackendListener', 'Summariser']
  if (listeners.includes(shortName)) return 'listener'
  
  // 控制器
  const controllers = ['LoopController', 'IfController', 'WhileController', 'ForeachController',
                       'OnceOnlyController', 'RandomController', 'TransactionController', 
                       'CriticalSectionController', 'ThroughputController', 'SwitchController',
                       'InterleaveController', 'RandomOrderController', 'RunTime']
  if (controllers.includes(shortName)) return 'controller'
  
  return 'other'
}

/**
 * 获取指定类别的所有元素
 * @param {string} category - 类别名称
 * @returns {Array} 元素数组 [{type, label, icon}, ...]
 */
export function getElementsByCategory(category) {
  const elements = []
  
  Object.keys(ELEMENT_META).forEach(type => {
    const meta = ELEMENT_META[type]
    const elementCategory = meta.category || getElementCategory(type)
    
    if (elementCategory === category) {
      elements.push({
        type,
        label: meta.label,
        icon: meta.icon || 'QuestionFilled'
      })
    }
  })
  
  return elements
}

/**
 * 判断元素是否为叶子节点（不能有子元素）
 * @param {string} testclass - JMeter 元素类型
 * @returns {boolean} 是否为叶子节点
 */
export function isLeafElement(testclass) {
  const meta = getElementMeta(testclass)
  return meta ? meta.isLeaf === true : true // 未知元素默认为叶子
}

/**
 * 获取元素的摘要信息
 * @param {string} testclass - JMeter 元素类型
 * @param {object} properties - 元素属性对象
 * @returns {string} 摘要字符串
 */
export function getElementSummary(testclass, properties) {
  const meta = getElementMeta(testclass)
  if (!meta) return ''
  
  // 优先使用 summary 函数
  if (meta.summary && typeof meta.summary === 'function') {
    return meta.summary(properties)
  }
  
  // 使用 summaryKeys 生成摘要
  if (!meta.summaryKeys || meta.summaryKeys.length === 0) return ''
  
  // 特殊处理 HTTP Sampler：组合 method + domain + path
  if (testclass === 'HTTPSamplerProxy') {
    const method = properties['HTTPSampler.method'] || 'GET'
    const domain = properties['HTTPSampler.domain'] || ''
    const path = properties['HTTPSampler.path'] || ''
    if (domain || path) return `${method} ${domain}${path}`
    return ''
  }
  
  // 通用处理：用 " | " 连接非空值
  const parts = meta.summaryKeys
    .map(sk => {
      const val = properties[sk.key]
      if (!val && val !== 0) return null
      const display = String(val).length > 30 ? String(val).substring(0, 30) + '...' : String(val)
      return sk.label ? `${sk.label}: ${display}` : display
    })
    .filter(Boolean)
  return parts.join(' | ')
}

// 导出 ELEMENT_META 供外部使用
export { ELEMENT_META }

const PROP_TAG_NAMES = ['stringProp', 'intProp', 'boolProp', 'longProp', 'floatProp', 'doubleProp']

// ============================================
// JMX 解析函数
// ============================================

/**
 * 生成唯一ID
 * @returns {string}
 */
function generateId() {
  return 'node_' + Math.random().toString(36).substr(2, 9) + '_' + Date.now()
}

/**
 * 解析属性值
 * @param {Element} propNode - 属性节点
 * @returns {string|number|boolean}
 */
function parsePropValue(propNode) {
  const tagName = propNode.tagName
  const textContent = propNode.textContent || ''
  
  switch (tagName) {
    case 'boolProp':
      return textContent === 'true'
    case 'intProp':
    case 'longProp':
      return parseInt(textContent, 10) || 0
    case 'floatProp':
    case 'doubleProp':
      return parseFloat(textContent) || 0
    case 'stringProp':
    default:
      return textContent
  }
}

/**
 * 解析 elementProp 节点
 * @param {Element} elementPropNode - elementProp 节点
 * @returns {object} 解析后的属性对象
 */
function parseElementProp(elementPropNode) {
  const result = {}
  const elementType = elementPropNode.getAttribute('elementType') || ''
  
  // 递归解析子属性
  const propNodes = elementPropNode.children
  for (let i = 0; i < propNodes.length; i++) {
    const child = propNodes[i]
    const name = child.getAttribute('name')
    
    if (name) {
      if (child.tagName === 'elementProp') {
        // 嵌套的 elementProp
        const nestedName = child.getAttribute('name')
        if (nestedName) {
          result[nestedName] = parseElementProp(child)
        }
      } else if (child.tagName === 'collectionProp') {
        // 集合属性
        result[name] = parseCollectionProp(child)
      } else {
        result[name] = parsePropValue(child)
      }
    }
  }
  
  return result
}

/**
 * 解析 collectionProp 节点
 * @param {Element} collectionNode - collectionProp 节点
 * @returns {Array} 解析后的数组
 */
function parseCollectionProp(collectionNode) {
  const items = []
  const children = collectionNode.children
  
  for (let i = 0; i < children.length; i++) {
    const child = children[i]
    if (child.tagName === 'elementProp') {
      items.push(parseElementProp(child))
    } else if (child.tagName === 'stringProp') {
      // 直接的 stringProp 子元素（如 ResponseAssertion 的 Asserion.test_strings）
      items.push(child.textContent || '')
    } else if (child.tagName === 'collectionProp') {
      // 嵌套的 collectionProp（如 UltimateThreadGroup 的数据）
      items.push(parseCollectionProp(child))
    }
  }
  
  return items
}

/**
 * 从 elementProp 中提取扁平化属性
 * @param {Element} parentNode - 父节点
 * @param {string} elementPropName - elementProp 的 name 属性
 * @returns {object} 提取的属性
 */
function extractElementPropProperties(parentNode, elementPropName) {
  const properties = {}
  const elementProps = parentNode.querySelectorAll(`elementProp[name="${elementPropName}"]`)
  
  for (let i = 0; i < elementProps.length; i++) {
    const prop = elementProps[i]
    const parsed = parseElementProp(prop)
    
    // 扁平化属性
    Object.keys(parsed).forEach(key => {
      const value = parsed[key]
      if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
        // 嵌套对象，添加前缀
        Object.keys(value).forEach(nestedKey => {
          properties[`${key}.${nestedKey}`] = value[nestedKey]
        })
      } else {
        properties[key] = value
      }
    })
  }
  
  return properties
}

/**
 * 解析单个 JMeter 元素
 * @param {Element} elementNode - JMeter 元素节点
 * @returns {object} 解析后的节点对象
 */
function parseJMeterElement(elementNode) {
  const testclass = elementNode.getAttribute('testclass') || ''
  const testname = elementNode.getAttribute('testname') || ''
  const enabled = elementNode.getAttribute('enabled') !== 'false'
  
  const properties = {}
  
  // 解析直接子属性 (stringProp, intProp, boolProp, longProp)
  const propNodes = elementNode.children
  for (let i = 0; i < propNodes.length; i++) {
    const child = propNodes[i]
    const tagName = child.tagName
    const name = child.getAttribute('name')
    
    if (!name) continue
    
    if (tagName === 'stringProp' || tagName === 'intProp' || 
        tagName === 'boolProp' || tagName === 'longProp' ||
        tagName === 'floatProp' || tagName === 'doubleProp') {
      properties[name] = parsePropValue(child)
    } else if (tagName === 'elementProp') {
      // elementProp 可能包含嵌套属性或集合
      const elementType = child.getAttribute('elementType') || ''
      const nestedProps = parseElementProp(child)
      
      // 特殊处理：如果包含 collectionProp，保留数组
      // 否则扁平化属性
      const hasCollection = Array.isArray(Object.values(nestedProps).find(v => Array.isArray(v)))
      
      if (hasCollection) {
        // 保留原始结构用于特殊处理
        properties[name] = nestedProps
        
        // 同时扁平化到 properties 以便编辑
        Object.keys(nestedProps).forEach(key => {
          const value = nestedProps[key]
          if (Array.isArray(value)) {
            properties[key] = value
          } else if (typeof value === 'object' && value !== null) {
            Object.keys(value).forEach(nestedKey => {
              properties[`${key}.${nestedKey}`] = value[nestedKey]
            })
          }
        })
      } else {
        // 扁平化嵌套属性
        Object.keys(nestedProps).forEach(key => {
          const value = nestedProps[key]
          if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
            Object.keys(value).forEach(nestedKey => {
              properties[`${key}.${nestedKey}`] = value[nestedKey]
            })
          } else {
            properties[key] = value
          }
        })
      }
    } else if (tagName === 'collectionProp') {
      // 特殊处理 UltimateThreadGroup 的 ultimatethreadgroupdata
      if (name === 'ultimatethreadgroupdata') {
        const rows = []
        child.querySelectorAll(':scope > collectionProp').forEach(rowProp => {
          const values = []
          rowProp.querySelectorAll('stringProp').forEach(sp => {
            values.push(sp.textContent || '0')
          })
          if (values.length >= 5) {
            rows.push({
              threads: values[0],
              initialDelay: values[1],
              startupTime: values[2],
              holdTime: values[3],
              shutdownTime: values[4]
            })
          }
        })
        properties[name] = rows
      } else {
        properties[name] = parseCollectionProp(child)
      }
    }
  }

  if ((testclass.includes('.') ? testclass.split('.').pop() : testclass) === 'HTTPSamplerProxy') {
    properties['HTTPSampler.body'] = getHttpBody(properties)
  }
  
  return {
    id: generateId(),
    testclass,
    testname,
    enabled,
    properties,
    children: [],
    _xmlNode: elementNode
  }
}

/**
 * 递归解析 hashTree 结构
 * @param {Element} hashTreeNode - hashTree 节点
 * @returns {Array} 子节点数组
 */
function parseHashTree(hashTreeNode) {
  const children = []
  const childNodes = hashTreeNode.children
  
  let i = 0
  while (i < childNodes.length) {
    const node = childNodes[i]
    
    if (node.tagName === 'hashTree') {
      // hashTree 节点本身跳过，它应该紧跟在元素后面处理
      i++
      continue
    }
    
    // 解析当前元素
    const element = parseJMeterElement(node)
    
    // 查找紧跟的 hashTree（包含子元素）
    i++
    if (i < childNodes.length && childNodes[i].tagName === 'hashTree') {
      element.children = parseHashTree(childNodes[i])
      i++
    }
    
    children.push(element)
  }
  
  return children
}

/**
 * 解析 JMX XML 字符串为树形结构
 * @param {string} xmlString - JMX XML 字符串
 * @returns {Array} 树形节点数组
 */
export function parseJMX(xmlString) {
  try {
    const parser = new DOMParser()
    const doc = parser.parseFromString(xmlString, 'text/xml')
    
    // 检查解析错误
    const parserError = doc.querySelector('parsererror')
    if (parserError) {
      throw new Error('XML 解析错误: ' + parserError.textContent)
    }
    
    const root = doc.documentElement
    
    // 验证根元素
    if (root.tagName !== 'jmeterTestPlan') {
      throw new Error('无效的 JMX 文件: 根元素不是 jmeterTestPlan')
    }
    
    // 查找根 hashTree
    const rootHashTree = root.querySelector(':scope > hashTree')
    if (!rootHashTree) {
      throw new Error('无效的 JMX 文件: 未找到根 hashTree')
    }
    
    const result = parseHashTree(rootHashTree)
    
    // 处理边界情况：确保 TestPlan 是唯一根节点
    // JMX 标准结构：根 hashTree 下应该只有一个 TestPlan
    // TestPlan 后面紧跟的 hashTree 包含它的子元素（线程组等）
    if (result.length > 1) {
      // 查找 TestPlan 节点
      const testPlanIndex = result.findIndex(n => n.testclass === 'TestPlan')
      
      if (testPlanIndex >= 0) {
        const testPlan = result[testPlanIndex]
        // 将其他顶层元素合并到 TestPlan 的 children 中
        // 这些元素应该是 TestPlan 的子元素，但由于某种原因被错误地放在了顶层
        result.forEach((node, idx) => {
          if (idx !== testPlanIndex && node.testclass !== 'TestPlan') {
            // 避免重复添加
            if (!testPlan.children.find(c => c.id === node.id)) {
              testPlan.children.push(node)
            }
          }
        })
        // 只返回 TestPlan
        return [testPlan]
      }
    }
    
    // 如果没有找到 TestPlan，但有多个元素，创建一个虚拟根节点
    if (result.length > 1 && !result.find(n => n.testclass === 'TestPlan')) {
      const virtualRoot = {
        id: generateId(),
        testclass: 'TestPlan',
        testname: 'Test Plan',
        enabled: true,
        properties: {},
        children: result,
        _xmlNode: null
      }
      return [virtualRoot]
    }
    
    return result
  } catch (error) {
    console.error('解析 JMX 失败:', error)
    throw error
  }
}

// ============================================
// JMX 序列化函数
// ============================================

/**
 * 创建属性节点
 * @param {Document} doc - XML 文档
 * @param {string} tagName - 标签名
 * @param {string} name - name 属性
 * @param {any} value - 值
 * @returns {Element} 创建的节点
 */
function createPropNode(doc, tagName, name, value) {
  const node = doc.createElement(tagName)
  node.setAttribute('name', name)
  node.textContent = String(value)
  return node
}

/**
 * 获取属性对应的标签名
 * @param {any} value - 属性值
 * @returns {string} 标签名
 */
function getPropTagName(value) {
  if (typeof value === 'boolean') return 'boolProp'
  if (typeof value === 'number') {
    if (Number.isInteger(value)) return 'intProp'
    return 'floatProp'
  }
  return 'stringProp'
}

function findDirectChild(parentNode, predicate) {
  return Array.from(parentNode.children).find(predicate) || null
}

function findDirectNamedChild(parentNode, tagName, name) {
  return findDirectChild(
    parentNode,
    (child) => child.tagName === tagName && child.getAttribute('name') === name
  )
}

function removeDirectPropNodes(parentNode, name) {
  Array.from(parentNode.children).forEach((child) => {
    if (PROP_TAG_NAMES.includes(child.tagName) && child.getAttribute('name') === name) {
      parentNode.removeChild(child)
    }
  })
}

/**
 * 更新或创建属性节点
 * @param {Element} parentNode - 父节点
 * @param {string} name - 属性名
 * @param {any} value - 属性值
 */
function updateOrCreateProp(parentNode, name, value) {
  // 查找现有节点
  const tagName = getPropTagName(value)
  const existingNode = findDirectChild(
    parentNode,
    (child) => PROP_TAG_NAMES.includes(child.tagName) && child.getAttribute('name') === name
  )
  
  if (existingNode) {
    // 如果标签类型改变，需要重新创建
    if (existingNode.tagName !== tagName) {
      const newNode = createPropNode(parentNode.ownerDocument, tagName, name, value)
      parentNode.insertBefore(newNode, existingNode)
      parentNode.removeChild(existingNode)
    } else {
      existingNode.textContent = String(value)
    }
  } else {
    // 创建新节点
    const newNode = createPropNode(parentNode.ownerDocument, tagName, name, value)
    parentNode.appendChild(newNode)
  }
}

/**
 * 创建键值对列表的 collectionProp
 * @param {Document} doc - XML 文档
 * @param {string} collectionName - 集合名
 * @param {Array} itemKeys - 键列表
 * @param {Array} items - 数据项数组
 * @param {string} itemElementType - 集合项的 elementType
 * @returns {Element} collectionProp 节点
 */
function createKeyValueCollectionProp(doc, collectionName, itemKeys, items, itemElementType = 'Header') {
  const collectionProp = doc.createElement('collectionProp')
  collectionProp.setAttribute('name', collectionName)
  
  items.forEach(item => {
    const elementProp = doc.createElement('elementProp')
    elementProp.setAttribute('name', '')
    elementProp.setAttribute('elementType', itemElementType)
    
    itemKeys.forEach((key, index) => {
      const value = item[index] !== undefined ? item[index] : ''
      const stringProp = doc.createElement('stringProp')
      stringProp.setAttribute('name', key)
      stringProp.textContent = String(value)
      elementProp.appendChild(stringProp)
    })
    
    collectionProp.appendChild(elementProp)
  })
  
  return collectionProp
}

/**
 * 更新 elementProp 中的 collectionProp
 * @param {Element} parentNode - 父节点
 * @param {string} elementPropName - elementProp 名称
 * @param {string} elementType - elementType 属性
 * @param {string} collectionName - collectionProp 名称
 * @param {Array} itemKeys - 键列表
 * @param {Array} items - 数据项数组
 * @param {string} itemElementType - 集合项的 elementType
 */
function updateElementPropCollection(
  parentNode,
  elementPropName,
  elementType,
  collectionName,
  itemKeys,
  items,
  itemElementType = 'Header'
) {
  const doc = parentNode.ownerDocument
  
  // 查找或创建 elementProp
  let elementProp = findDirectNamedChild(parentNode, 'elementProp', elementPropName)
  if (!elementProp) {
    elementProp = doc.createElement('elementProp')
    elementProp.setAttribute('name', elementPropName)
    elementProp.setAttribute('elementType', elementType)
    parentNode.appendChild(elementProp)
  }
  
  // 查找或创建 collectionProp
  let collectionProp = findDirectNamedChild(elementProp, 'collectionProp', collectionName)
  if (collectionProp) {
    elementProp.removeChild(collectionProp)
  }
  
  // 创建新的 collectionProp
  collectionProp = createKeyValueCollectionProp(doc, collectionName, itemKeys, items, itemElementType)
  elementProp.appendChild(collectionProp)
}

function updateHttpBodyElementProp(elementNode, body) {
  const doc = elementNode.ownerDocument

  removeDirectPropNodes(elementNode, 'HTTPSampler.body')

  let elementProp = findDirectNamedChild(elementNode, 'elementProp', 'HTTPsampler.Arguments')
  if (!elementProp) {
    elementProp = doc.createElement('elementProp')
    elementProp.setAttribute('name', 'HTTPsampler.Arguments')
    elementProp.setAttribute('elementType', 'Arguments')
    elementNode.appendChild(elementProp)
  }

  let collectionProp = findDirectNamedChild(elementProp, 'collectionProp', 'Arguments.arguments')
  if (collectionProp) {
    elementProp.removeChild(collectionProp)
  }

  collectionProp = doc.createElement('collectionProp')
  collectionProp.setAttribute('name', 'Arguments.arguments')

  const argumentElement = doc.createElement('elementProp')
  argumentElement.setAttribute('name', '')
  argumentElement.setAttribute('elementType', 'HTTPArgument')

  argumentElement.appendChild(createPropNode(doc, 'boolProp', 'HTTPArgument.always_encode', false))
  argumentElement.appendChild(createPropNode(doc, 'stringProp', 'Argument.value', body || ''))
  argumentElement.appendChild(createPropNode(doc, 'stringProp', 'Argument.metadata', '='))

  collectionProp.appendChild(argumentElement)
  elementProp.appendChild(collectionProp)
}

function clearChildren(node) {
  while (node.firstChild) {
    node.removeChild(node.firstChild)
  }
}

function createElementNode(doc, node) {
  if (node._xmlNode) {
    return doc.importNode(node._xmlNode, true)
  }

  const testclass = node.testclass === 'UltimateThreadGroup'
    ? 'kg.apc.jmeter.threads.UltimateThreadGroup'
    : node.testclass

  const elementNode = doc.createElement(testclass)
  elementNode.setAttribute('testclass', testclass)
  elementNode.setAttribute('guiclass', `${testclass.includes('.') ? testclass.split('.').pop() : testclass}Gui`)
  return elementNode
}

/**
 * 根据 ID 查找树节点
 * @param {Array} tree - 树数组
 * @param {string} id - 节点ID
 * @returns {object|null}
 */
function findNodeById(tree, id) {
  for (const node of tree) {
    if (node.id === id) return node
    if (node.children && node.children.length > 0) {
      const found = findNodeById(node.children, id)
      if (found) return found
    }
  }
  return null
}

/**
 * 更新元素节点的属性（增量更新）
 * @param {Element} elementNode - 元素节点
 * @param {object} node - 树节点对象
 * @param {Document} doc - XML 文档
 */
function updateElementProperties(elementNode, node, doc) {
  const meta = getElementMeta(node.testclass)
  
  if (meta && meta.properties) {
    meta.properties.forEach(propDef => {
      const value = node.properties[propDef.key]
      if (value === undefined) return
      
      if (propDef.type === 'keyValueList') {
        // 键值对列表特殊处理
        const items = value || []
        const formattedItems = items.map(item => {
          if (Array.isArray(item)) return item
          return propDef.itemKeys.map(key => item[key] || '')
        })
        
        // 确定 elementType
        let elementType = 'Header'
        if (propDef.key === 'Arguments.arguments') elementType = 'Argument'
        
        updateElementPropCollection(
          elementNode,
          propDef.key.split('.')[0], // Arguments 或 HeaderManager
          elementType === 'Argument' ? 'Arguments' : elementType,
          propDef.key,
          propDef.itemKeys,
          formattedItems,
          elementType
        )
      } else if (propDef.type === 'threadSchedule') {
        // UltimateThreadGroup 线程调度配置序列化
        const rows = value || []
        
        // 查找或创建 collectionProp
        let collectionProp = findDirectNamedChild(elementNode, 'collectionProp', 'ultimatethreadgroupdata')
        if (collectionProp) {
          elementNode.removeChild(collectionProp)
        }
        
        // 创建新的 collectionProp
        collectionProp = doc.createElement('collectionProp')
        collectionProp.setAttribute('name', 'ultimatethreadgroupdata')
        
        // 添加每一行数据
        rows.forEach(row => {
          const rowProp = doc.createElement('collectionProp')
          rowProp.setAttribute('name', String(row.threads || '0'))
          
          // 5个 stringProp: 线程数, 初始延迟, 启动时间, 持续时间, 关闭时间
          const values = [
            String(row.threads || '0'),
            String(row.initialDelay || '0'),
            String(row.startupTime || '0'),
            String(row.holdTime || '0'),
            String(row.shutdownTime || '0')
          ]
          
          values.forEach(val => {
            const stringProp = doc.createElement('stringProp')
            stringProp.setAttribute('name', '')
            stringProp.textContent = val
            rowProp.appendChild(stringProp)
          })
          
          collectionProp.appendChild(rowProp)
        })
        
        elementNode.appendChild(collectionProp)
      } else if (propDef.type === 'stringList') {
        // 字符串列表（如 ResponseAssertion 的 test_strings）
        const strings = Array.isArray(value) ? value : (value ? [value] : [])
        
        // 查找或创建 collectionProp
        let collectionProp = findDirectNamedChild(elementNode, 'collectionProp', propDef.key)
        if (collectionProp) {
          elementNode.removeChild(collectionProp)
        }
        
        collectionProp = doc.createElement('collectionProp')
        collectionProp.setAttribute('name', propDef.key)
        
        strings.forEach(str => {
          const stringProp = doc.createElement('stringProp')
          // JMeter 使用随机整数作为 name，这里生成类似的值
          stringProp.setAttribute('name', String(Math.floor(Math.random() * 2147483647)))
          stringProp.textContent = String(str)
          collectionProp.appendChild(stringProp)
        })
        
        elementNode.appendChild(collectionProp)
      } else if (propDef.special === 'httpBody') {
        updateHttpBodyElementProp(elementNode, value)
      } else if (propDef.nested) {
        // 嵌套属性
        const parts = propDef.key.split('.')
        const nestedElementName = propDef.nested
        const nestedPropName = parts.join('.')
        
        // 查找或创建嵌套的 elementProp
        let nestedElement = findDirectNamedChild(elementNode, 'elementProp', nestedElementName)
        if (!nestedElement) {
          nestedElement = doc.createElement('elementProp')
          nestedElement.setAttribute('name', nestedElementName)
          nestedElement.setAttribute('elementType', parts[0]) // LoopController
          elementNode.appendChild(nestedElement)
        }
        
        updateOrCreateProp(nestedElement, nestedPropName, value)
      } else {
        // 普通属性
        updateOrCreateProp(elementNode, propDef.key, value)
      }
    })
  } else {
    // 没有元数据定义，直接更新所有属性
    Object.keys(node.properties).forEach(key => {
      const value = node.properties[key]
      if (value === null || ['string', 'number', 'boolean'].includes(typeof value)) {
        updateOrCreateProp(elementNode, key, value)
      }
    })
  }
}

/**
 * 递归序列化树节点到 DOM（增量更新版本）
 * @param {Element} parentHashTree - 父 hashTree 节点
 * @param {Array} tree - 树节点数组
 * @param {Document} doc - XML 文档
 */
function serializeTreeToDOM(parentHashTree, tree, doc) {
  // 收集现有的子元素和对应的 hashTree
  const existingElements = []
  const children = Array.from(parentHashTree.children)
  for (let i = 0; i < children.length; i++) {
    const child = children[i]
    if (child.tagName !== 'hashTree') {
      // 记录元素和它后面的 hashTree
      const nextSibling = children[i + 1]
      const hashTree = (nextSibling && nextSibling.tagName === 'hashTree') ? nextSibling : null
      existingElements.push({
        element: child,
        hashTree: hashTree,
        testname: child.getAttribute('testname') || ''
      })
    }
  }
  
  // 清空 parentHashTree，稍后重新添加
  clearChildren(parentHashTree)
  
  tree.forEach((node, index) => {
    // 尝试找到对应的现有元素（通过原始节点引用或 testname）
    const existing = existingElements.find(e => 
      e.element === node._xmlNode || 
      (e.testname === node.testname && index < existingElements.length)
    )
    
    let elementNode
    let childHashTree = null
    
    if (existing && node._xmlNode) {
      // 使用现有节点（增量更新）
      elementNode = existing.element
      childHashTree = existing.hashTree
      
      // 更新基本属性
      elementNode.setAttribute('testname', node.testname)
      elementNode.setAttribute('enabled', String(node.enabled))
      
      // 增量更新属性（只更新定义的属性，保留其他原始属性）
      updateElementProperties(elementNode, node, doc)
    } else {
      // 创建新节点
      elementNode = createElementNode(doc, node)
      
      // 更新基本属性
      elementNode.setAttribute('testname', node.testname)
      elementNode.setAttribute('enabled', String(node.enabled))
      
      // 更新属性
      updateElementProperties(elementNode, node, doc)
    }
    
    // 添加元素节点
    parentHashTree.appendChild(elementNode)
    
    // 处理 hashTree
    if (!childHashTree) {
      childHashTree = doc.createElement('hashTree')
    }
    parentHashTree.appendChild(childHashTree)

    // 递归处理子节点
    if (node.children && node.children.length > 0) {
      serializeTreeToDOM(childHashTree, node.children, doc)
    } else {
      // 如果没有子节点，清空 hashTree
      clearChildren(childHashTree)
    }
  })
}

/**
 * 将修改后的树结构序列化回 XML 字符串
 * @param {Array} tree - 树形节点数组
 * @param {string} originalXmlString - 原始 XML 字符串
 * @returns {string} 序列化后的 XML 字符串
 */
export function serializeJMX(tree, originalXmlString) {
  try {
    const parser = new DOMParser()
    const doc = parser.parseFromString(originalXmlString, 'text/xml')
    
    // 检查解析错误
    const parserError = doc.querySelector('parsererror')
    if (parserError) {
      throw new Error('原始 XML 解析错误: ' + parserError.textContent)
    }
    
    const root = doc.documentElement
    const rootHashTree = root.querySelector(':scope > hashTree')
    
    if (!rootHashTree) {
      throw new Error('无效的 JMX 文件: 未找到根 hashTree')
    }
    
    // 序列化树结构到 DOM
    serializeTreeToDOM(rootHashTree, tree, doc)
    
    // 序列化回字符串
    const serializer = new XMLSerializer()
    return serializer.serializeToString(doc)
  } catch (error) {
    console.error('序列化 JMX 失败:', error)
    throw error
  }
}

// ============================================
// 辅助函数
// ============================================

/**
 * 从属性中提取 HTTP Body（特殊处理）
 * @param {object} properties - 节点属性对象
 * @returns {string} HTTP Body 内容
 */
export function getHttpBody(properties) {
  // 检查是否有原始 body 标记（类型兼容：boolean true 或字符串 "true"）
  const isRawBody = properties['HTTPSampler.postBodyRaw']
  if (isRawBody !== true && isRawBody !== 'true') {
    return ''
  }
  
  // 方式1：从扁平化的 Arguments.arguments 数组提取
  const argsArray = properties['Arguments.arguments']
  if (argsArray && Array.isArray(argsArray) && argsArray.length > 0) {
    const firstArg = argsArray[0]
    if (typeof firstArg === 'object' && firstArg !== null) {
      return firstArg['Argument.value'] || firstArg.value || ''
    }
  }
  
  // 方式2：从嵌套的 HTTPsampler.Arguments 结构提取
  const httpArgs = properties['HTTPsampler.Arguments']
  if (httpArgs && typeof httpArgs === 'object') {
    // 可能是 { 'Arguments.arguments': [...] } 结构
    const nestedArgs = httpArgs['Arguments.arguments']
    if (nestedArgs && Array.isArray(nestedArgs) && nestedArgs.length > 0) {
      const firstArg = nestedArgs[0]
      if (typeof firstArg === 'object' && firstArg !== null) {
        return firstArg['Argument.value'] || firstArg.value || ''
      }
    }
  }
  
  // 方式3：直接查找 body 属性
  return properties['HTTPSampler.body'] || properties['HTTPsampler.body'] || ''
}

/**
 * 设置 HTTP Body
 * @param {object} properties - 节点属性对象
 * @param {string} body - Body 内容
 */
export function setHttpBody(properties, body) {
  // 设置原始 body 标记（使用 boolean true 以保持类型一致性）
  properties['HTTPSampler.postBodyRaw'] = true
  
  // 设置 body 属性（用于编辑）
  properties['HTTPSampler.body'] = body
  
  // 构建扁平化的 Arguments.arguments 数组（用于 getHttpBody 方式1）
  properties['Arguments.arguments'] = [
    {
      'Argument.value': body,
      'Argument.name': '',
      'Argument.metadata': '=',
      'HTTPArgument.always_encode': false
    }
  ]
  
  // 同时构建嵌套的 HTTPsampler.Arguments 结构（用于 getHttpBody 方式2）
  properties['HTTPsampler.Arguments'] = {
    'Arguments.arguments': properties['Arguments.arguments']
  }
}

/**
 * 从 collectionProp 解析键值对列表
 * @param {object} properties - 节点属性对象
 * @param {string} collectionKey - 集合键名
 * @param {Array} itemKeys - 项目键列表
 * @returns {Array} 键值对数组，每项为 [value1, value2, ...]
 */
export function parseKeyValueList(properties, collectionKey, itemKeys) {
  const collection = properties[collectionKey]
  
  if (!collection || !Array.isArray(collection)) {
    return []
  }
  
  return collection.map(item => {
    return itemKeys.map(key => {
      if (item && item[key] !== undefined) {
        return String(item[key])
      }

      // 支持嵌套键名如 "Header.name"
      const parts = key.split('.')
      let value = item
      for (const part of parts) {
        value = value && value[part] !== undefined ? value[part] : ''
      }
      return String(value)
    })
  })
}

/**
 * 将键值对列表写回 properties
 * @param {object} properties - 节点属性对象
 * @param {string} collectionKey - 集合键名
 * @param {Array} itemKeys - 项目键列表
 * @param {Array} items - 键值对数组，每项为 [value1, value2, ...] 或对象
 */
export function setKeyValueList(properties, collectionKey, itemKeys, items) {
  // 转换为标准格式
  const formattedItems = items.map(item => {
    if (Array.isArray(item)) {
      const obj = {}
      itemKeys.forEach((key, index) => {
        obj[key] = item[index] || ''
      })
      return obj
    }
    return item
  })
  
  properties[collectionKey] = formattedItems
}

// ============================================
// CSVDataSet 引用文件提取
// ============================================

/**
 * 从 JMX 树中提取所有 CSVDataSet 引用的文件名
 * @param {Array} tree - parseJMX 返回的树形结构
 * @returns {Array} 引用的文件名数组（去重）
 */
export function extractCSVDataSetFiles(tree) {
  const files = new Set()
  
  function traverse(nodes) {
    for (const node of nodes) {
      if (node.testclass === 'CSVDataSet' && node.properties) {
        const filename = node.properties['filename']
        if (filename && filename.trim()) {
          files.add(filename.trim())
        }
      }
      if (node.children && node.children.length > 0) {
        traverse(node.children)
      }
    }
  }
  
  traverse(tree)
  return Array.from(files)
}

/**
 * 从 JMX XML 字符串中提取所有 CSVDataSet 引用的文件名
 * @param {string} xmlString - JMX XML 字符串
 * @returns {Array} 引用的文件名数组（去重）
 */
export function extractCSVDataSetFilesFromXML(xmlString) {
  if (!xmlString) return []
  
  try {
    const tree = parseJMX(xmlString)
    return extractCSVDataSetFiles(tree)
  } catch (error) {
    console.error('提取 CSVDataSet 文件失败:', error)
    return []
  }
}

// ============================================
// 默认导出
// ============================================

export default {
  getElementMeta,
  getElementCategory,
  getElementsByCategory,
  isLeafElement,
  getElementSummary,
  parseJMX,
  serializeJMX,
  getHttpBody,
  setHttpBody,
  parseKeyValueList,
  setKeyValueList,
  extractCSVDataSetFiles,
  extractCSVDataSetFilesFromXML,
  ELEMENT_META
}
